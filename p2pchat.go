/**********************************************************************/
/* P2P CHAT ROOM                         Author: Xingchi             **/
/* TeamMembers: Prarav Bagree, Ryan Culter, Nicolas Mellis           **/
/*                                                                   **/
/* This program is used to build a p2p communication network         **/
/* in a local environment. Given the knowledge of address informatio **/
/* of the roomhost, every new commer will be able to join the room.  **/
/* The room size is scalable.                                        **/
/* It's fine when a peer voluntarily or abnormally leaves the room.  **/
/* ********************************************************************/



/* NOTES!!

	1. for local chat, every node does not maintain conection to himself, so roomhost cannot make a enter room "JOIN" request
	2. sending error has been dealt with : EOF error--> connection lost(disconnect, free resources)
	3. When a peer abnormally crash......all informatiom about this out-of-date user will be deleted
	4. every node has 3 connections with the roomhost(one additional connection is built for initial JOIN request, I 
		donot want to torn it down later because this involves many error handling operations and leave it alive doesnot affect anything),
		every normal node share 2 connections with the other normal node 
	5. QML(UI interface) has not been added yet, but all backend APIs are already available.
	6. Un-dealt cases: @ what if the room host crashed or left? transfer the duty or destroy the room?
*/


package main

import(
	"fmt"
	"os"
	"strings"
	"sync"
	"bufio" 
	"net"
	"encoding/gob"
	"io"
)

//for temporary use only!
var users map[string]string

//a map of all connections map["username"]net.Conn
var connectionList map[string]string
var encoderList map[string]*gob.Encoder
var myUsrName string
var myIP string
var myPort string
var myAddr string
var mutex=new(sync.Mutex)

var roomHostName string
var roomHostIP string

// start connections, listen on server port, and collect user input
func main() {
	initSettings()

	fmt.Println(users[myUsrName])
	go listen()
	interaction()
	
}

type Message struct {
	Msgtype string
	Username string  // sender's username
	MyAddr string   //sender's ip
	MsgContent string  
	Usernames []string
	Addrs []string
	//Ports[] string
}

/*
*  get message from use input
*  send out to all connected users
*/
func interaction() {
	//message :=new(Message)
	

	reader:= bufio.NewReader(os.Stdin)

	for{
		line, err :=reader.ReadString('\n')
		
		if err!=nil {
			fmt.Println("read user input error!")
			break
		}

		line=strings.Trim(line," \n")
		parm:=strings.Split(line," ")

		if parm[0]=="exit" {
			//disconnect and break
			sayGoodBye()
			break
		}else if parm[0]=="conn" {

			//room host cannot send "JOIN" request
			fmt.Println(parm)
			if myUsrName==roomHostName || len(parm)!=4{
				continue
			}
			
			introduceMyself(parm[1],parm[2],parm[3])
		}else {
			//send the message out
			fmt.Println(line)
			msg:=createMsg("MESSAGE",myUsrName,myIP, line,make([]string,2),make([]string,2))
			

			msg.sendToAll()
		}
	}
	os.Exit(1)
}

/*
* Listen on the port and accept connections
*/
func listen() {

	listener,err:=net.Listen("tcp",":"+myPort)

	if err!=nil{
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println(listener.Addr().String())
	for{
		newCon, err:=listener.Accept()

		if err!=nil {
			continue
		}

		//go reveive msg from this connection
		fmt.Println("get connection from", newCon.RemoteAddr().String())
		go recv(newCon)
	}
}
/*
* Initialize all the connections
*/
func initSettings(){
	/*initial sys settings part*/
	roomHostName="Bob"
	roomHostIP="localhost"

	getMyIP()
	connectionList=make(map[string]string)
	encoderList=make(map[string]*gob.Encoder)
	myUsrName = os.Args[1]
	myPort=os.Args[2]
	myAddr=myIP+":"+myPort
	/*initial sys settings part*/

}

/*
* keep receiving messages from the Accpted Conn 
*/

func recv(conn net.Conn){
	defer conn.Close()

	decoder:=gob.NewDecoder(conn)

	//msg:=new(Message)
	var msg Message
	var sender string

	for{
		
		if err:=decoder.Decode(&msg);err!=nil{
			//fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
			fmt.Printf("This connection torn down~\nRemote: %s\nLocal: %s\n",
				conn.RemoteAddr().String(),conn.LocalAddr().String())
			handleLEVE(sender)

			return
		}
		//fmt.Printf("Receive message: %+v \n",&msg)
		sender=msg.Username

		switch msg.Msgtype{
			case "MESSAGE": 
				fmt.Printf("Receive Msg From %s: %s\n",msg.Username,msg.MsgContent)

			//only super node can receive "JOIN" message
			case "JOIN":
				if !handleJOIN(&msg,conn){
					fmt.Println("Handle connection request error!",conn.RemoteAddr().String())
					return
				}

			//be asked to build connection(s) to teammates
			case "ADD" :
				addPeers(&msg)

			//notified that some one has left
			case "LEAVE": 
				handleLEVE(msg.Username)

		}
	}
}

func createConn(targetAddr string,targetName string) (net.Conn,error){
	//var targetPort string= users[targetName]

	//targetAddr:=IP+":"+targetPort
	conn,err:=net.Dial("tcp",targetAddr)

	if err!=nil{
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		fmt.Fprintf(os.Stderr,"this guy[%s] dose not exist \n",targetAddr)
		return nil,err
	}

	mutex.Lock()
	connectionList[targetName]=targetAddr
	encoderList[targetName]=gob.NewEncoder(conn)
	mutex.Unlock()

	return conn,err
}

/*
* send message to one user by username
*/
func (msg *Message)sendToOne (targetName string) error{

	if enc,ok:=encoderList[targetName]; ok {

		err:=enc.Encode(msg)
		

		if err!=nil{
			fmt.Println(err)
			if err==io.EOF{
				fmt.Fprintf(os.Stderr,"Connection with %s LOST!\n",targetName)
				handleLEVE(targetName)
			}
		}
		
		//fmt.Printf("Sent message: %+v\n",msg)
		
		return err
	}else{
		fmt.Println("The destination user doesnot exist...")
	}
	return nil
}

/*
* send one message to all connections
*/
func (msg *Message) sendToAll(){
	
	for usrname,_:=range encoderList{
		msg.sendToOne(usrname)
	}
}

/*
* create a Message with specified values
*/

func createMsg(kind string, myname string,myAd string, MSG string, Usernames []string, adrs []string)(msg *Message){
	msg = new(Message)
	msg.Msgtype = kind
	msg.Username = myname
	msg.MyAddr = myAd
	msg.MsgContent = MSG
	msg.Usernames = Usernames
	msg.Addrs = adrs
	return 
}

/*
* introduce myself to the rendezvous node--typically the creater of a room
*/

func introduceMyself(targetIP string,targetPort string,targetName string){
	conn,err:=createConn(targetIP+":"+targetPort,targetName)

	initialErr(err)

	//listmsg:=new(Message)

	jmsg:=createMsg("JOIN",myUsrName,myAddr,"",make([]string,0),make([]string,0))

	jmsg.sendToOne(targetName)

	go recv(conn)
}

/*
* get the ipaddr of myself
*/ 
func getMyIP(){
	name, err := os.Hostname()
	initialErr(err)
	//addr, err := net.LookupIP(name)
	addr, err := net.ResolveIPAddr("ip",name)
	initialErr(err)
	myIP = string(addr.String())

	return
}

/*
* deal with the error happens at initial time. 
* result: exit program
*/

func initialErr(err error){
	if err!=nil {
		fmt.Fprintf(os.Stderr,"Critical error happened when init this peer!\n")
		os.Exit(1)
	}
}

/*
* handle a "JOIN" request from a new peer
*/

func handleJOIN(msg *Message, conn net.Conn) bool{
	/*chaeck if th username already exist*/
	enc:=gob.NewEncoder(conn)

	if userExist(msg.Username) {
		warning := createMsg("MESSAGE",myUsrName,myAddr,
			"User name already taken, Choose another one!",make([]string,0),make([]string,0))
		
		
		enc.Encode(warning)

		return false
	}



	/* send ADD command back to tell the applicant to make connections to teammates*/
	teammates, addrs:=getFromMap(connectionList)
	teammates=append(teammates,myUsrName) // the new peer need to build another connection to the host.
	addrs=append(addrs,myAddr)

	if len(addrs)>0{
		ADDMsg:=createMsg("ADD",myUsrName,myAddr,"",teammates,addrs)
		enc.Encode(ADDMsg)
	}


	/*ask his teammates to build connections to him*/
	friendUsrName:=make([]string,1)
	friendIP:=make([]string,1)

	friendUsrName[0]=msg.Username
	friendIP[0]=msg.MyAddr
	AddFriendMsg:=createMsg("ADD",myUsrName,myAddr,"",friendUsrName,friendIP)
	AddFriendMsg.sendToAll()


	/*create a connection back to the applicant, add the connection to my map*/
	/*this part has tp be the last one, because I can not ask the new peer to connect to himself*/
	_,err:=createConn(msg.MyAddr,msg.Username) 
	if err!=nil {
		warningForConn := createMsg("MESSAGE",myUsrName,myAddr,
			"Make connection error!",make([]string,0),make([]string,0))
		enc.Encode(warningForConn)

		return false
	}

	fmt.Printf("Welcome %s\n",msg.Username)
	return true
}

/*
* get arrays of keys and vals from  a map
*/

func getFromMap(mp map[string]string)([]string, []string){
	var keys []string
	var vals []string

	for key,val :=range mp {
		keys=append(keys,key)
		vals=append(vals,val)
	}
	return keys,vals
}

/*
* if the connection to that username:ip is built
*/
func userExist(userName string) bool{
	_,ok := connectionList[userName]

	if ok {
		return true
	}
	return false
}

/*
* After get "ADD" msg, build connections to the list
*/
func addPeers(msg *Message){
	for i := 0; i < len(msg.Usernames); i++ {
		createConn(msg.Addrs[i],msg.Usernames[i])
	}
}

/*
* tell other peers that you are leaving
*/

func sayGoodBye(){
	byeMsg:=createMsg("LEAVE",myUsrName,myAddr,
			"",make([]string,0),make([]string,0))

	byeMsg.sendToAll()
}

/*
* when receive a LEAVE message
*/
func handleLEVE(username string) {
	if _,ok:=connectionList[username];ok {

		fmt.Printf("%s has Left the room\n",username)
		mutex.Lock()
		delete(connectionList,username)
		delete(encoderList,username)
		mutex.Unlock()
	}
}

