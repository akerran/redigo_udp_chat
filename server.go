package main

import (
    "net"
    "fmt"
    "github.com/gomodule/redigo/redis"
    "sync"
    "strings"
)

const MAXCHATHISTORY int = 20

func check_trim_history(c redis.Conn) {
    msgnum,err := redis.Int(c.Do("llen","messages"))
    if err != nil {
        fmt.Println("Failed to get chat history length:", err.Error())
    } else {
        if msgnum > MAXCHATHISTORY {
            _,err := c.Do("lpop", "messages")
            if err != nil {
                fmt.Println("Failed to remove old message:", err.Error())
            }
        }
    }
}

func db_append(c redis.Conn, username, msgid, dbrecord string) {
    _, err := c.Do("rpush", "messages", dbrecord)
    if err != nil {
        panic(err)
    }
    c.Do("hset", "user"+":"+username, msgid, dbrecord)
    if err != nil {
        panic(err)
    }
    check_trim_history(c)
}

func parse_msg(msg string) (username, msgid string) {
    msgid = msg[:6]
    i := strings.Index(msg, ":")
    username = msg[7:i]

    return username, msgid
}


// search message in hash for current client and remove found message
// from 'messages' list
func remove_message(c redis.Conn, user string, msgid string) {
    msg, err := redis.Bytes(c.Do("hget", "user:"+user, msgid))
    if err != nil {
        fmt.Println("Message not found", err)
    } else {
        c.Do("lrem", "messages", "0", string(msg))
        c.Do("hdel", "user:"+user, msgid)
        fmt.Println("Message removed")
    }
}

func client_listener(wg *sync.WaitGroup, c redis.Conn) {
    listenAddr,err := net.ResolveUDPAddr("udp4", ":7713")
    if err != nil {
        panic(err)
    }
    pc,err := net.ListenUDP("udp4", listenAddr)
    if err != nil {
        panic(err)
    }
    defer pc.Close()


    // broadcast connection
    bcastConn,err := net.ListenPacket("udp4", ":8813")
    if err != nil {
        panic(err)
    }
    defer bcastConn.Close()

    baddr,err := net.ResolveUDPAddr("udp4", "192.168.8.255:8813")
    if err != nil {
        panic(err)
    }

    buf := make([]byte, 1024)
        var history string
    for {
        n,clntAddr,err := pc.ReadFrom(buf)
        if err != nil {
            panic(err)
        }
        strmsg := string(buf[:n])
        fmt.Println("Got message:", clntAddr, strmsg)

        if strings.Contains(strmsg, "/h")  {
	    // we need to send history to exact client only
            // therefore get his IP and append broadcast port
	    ccaddr := clntAddr.String()
	    i := strings.Index(ccaddr, ":")
	    ccaddr = ccaddr[:i] + ":8813"
	    fmt.Println("client:", ccaddr) 
            caddr,err := net.ResolveUDPAddr("udp4", ccaddr)
            if err != nil {
                panic(err)
            }
            history = load_history(c)
            pc.WriteTo([]byte(history), caddr)
            continue
        }

        username,msgid := parse_msg(strmsg)
        if strings.Contains(strmsg, "/rm")  {
            remove_message(c, username, strings.TrimSpace(strmsg[3:]))
        } else {
            // got standard message
            // append it to history and send via broadcast to other clients
            db_append(c, username, msgid, strmsg)
            pc.WriteTo(buf, baddr)
        }
    }
    wg.Done()
}

// get contents from messages list as a chat history
func load_history(c redis.Conn) string {
    var s string
    values,err := redis.Values(c.Do("lrange", "messages",0,MAXCHATHISTORY))
    if err != nil {
        fmt.Println("Failed to load chat history:",err.Error())
    }
    for _,v := range values {
        s = s+string(v.([]byte))+"\n"
    }
    return s
}

func main() {
    redisConn, err := redis.Dial("tcp", ":6379")
    if err != nil {
        fmt.Println(err)
        return
    }
    defer redisConn.Close()

    wg := sync.WaitGroup{}
    wg.Add(1)
    go client_listener(&wg, redisConn)

    wg.Wait()
}
