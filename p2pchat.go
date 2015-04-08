package main

import(
	"fmt"
	"os"
	"strings"
//	"sync"
	"bufio" 
	"net"
	"encoding/gob"
	"binary"
)

//for temporary use only!
var users map[string]string

//a map of all connections map["username"]net.Conn
var connectionList map[string]net.Conn
var myUsrName string

// start connections, listen on server port, and collect user input
func main() {
	users=make(map[string]string)
	users["Bob"]=":9999"
	users["Alice"]=":9998"
	users["Lee"]="=:2190"
	users["Alex"]=":2191"

	connectionList=make(map[string]net.Conn)


	var myUsrName string= os.Args[1]
	fmt.Println(users[myUsrName])
	go listen(users[myUsrName])
	 interaction()
	
}

type Message struct {
	msgtype string
	username string  // sender's username
	IP string   //sender's ip
	msgContent string  
	Usernames []string
	IPs []string
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

		if line=="exit" {
			//disconnect and break
			break
		}else if line=="conn" {
			createConn("localhost","Bob")
		}else {
			//send the message out
			fmt.Println(line)
			msg:=createMsg("PUBLIC",myUsrName,"", line,make([]string,2),make([]string,2))

			msg.sendToAll()
		}
	}
	os.Exit(1)
}

/*
* Listen on the port and accept connections
*/
func listen(port string) {
	// tcpAddr,err:=net.ResolveTCPAddr("tcp4","localhost"+":"+port)
	// if err!=nil {
	// 	fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
	// 	os.Exit(1)
	// }

	listener,err:=net.Listen("tcp",port)

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
* keep receiving messages from the Accpted Conn 
*/

func recv(conn net.Conn){
	defer conn.Close()

	decoder:=gob.NewDecoder(conn)

	msg:=new(Message)

	for{
		if err:=decoder.Decode(msg);err!=nil{
			fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
			fmt.Println("Read message error")
			return
		}
		fmt.Println(msg)
		fmt.Println(msg.msgContent)
		switch msg.msgtype{
			case "MESSAGE": 
				fmt.Println(msg.msgContent)
		}
	}
}

func createConn(IP string, targetName string) (connection net.Conn) {
	var targetPort string= users[targetName]

	// tcpAddr, err:=net.ResolveTCPAddr("tcp",IP+targetPort)

	// if err!=nil{
	// 	fmt.Println("this guy dose not exist 1")
	// 	return nil
	// }

	conn,err:=net.Dial("tcp",IP+targetPort)

	if err!=nil{
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		fmt.Println("this guy dose not exist 3",IP+targetPort)
		return nil
	}

	connectionList[targetName]=conn

	return conn
}

/*
* send message to one user by username
*/
func (msg *Message) sendToOne (targetName string){

	if connection,ok:=connectionList[targetName]; ok {
		enc:=gob.NewEncoder(connection)

		fmt.Println("in send to one ",msg.msgContent)

		enc.Encode(msg)

		fmt.Println(binary.Size(connection))
	}else{
		fmt.Println("The destination user doesnot exist...")
	}
	
}

/*
* send one message to all connections
*/
func (msg *Message) sendToAll(){
	
	for usrname,_:=range connectionList{
		msg.sendToOne(usrname)
	}
}

/*
* create a Message with specified values
*/

func createMsg(kind string, myname string,myip string, MSG string, Usernames []string, IPs []string)(msg *Message){
	msg = new(Message)
	msg.msgtype = kind
	msg.username = myname
	msg.IP = myip
	msg.msgContent = MSG
	msg.Usernames = Usernames
	msg.IPs = IPs
	return 
}