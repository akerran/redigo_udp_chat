package main

import (
    "net"
    "fmt"
    "github.com/gomodule/redigo/redis"
    "sync"
    "strings"
    "time"
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

func remove_message(c redis.Conn, user string, msgid string) {
    msg, err := redis.Bytes(c.Do("hget", "user:"+user, msgid))
    if err != nil {
        fmt.Println("Message not found")
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

    buf := make([]byte, 1024)
    for {
        n,_,err := pc.ReadFrom(buf)
        if err != nil {
            panic(err)
        }
        strmsg := string(buf[:n])
        fmt.Println("Got message:", strmsg)
        username,msgid := parse_msg(strmsg)

        if strings.Contains(strmsg, "/rm")  {
            remove_message(c, username, strings.TrimSpace(strmsg[3:]))
        } else {
            db_append(c, username, msgid, strmsg)
        }
    }
    wg.Done()
}



// func count_clients() {
//     c.Do("client", "list")
// }

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

    bcastConn,err := net.ListenPacket("udp4", ":8813")
    if err != nil {
        panic(err)
    }
    defer bcastConn.Close()

    baddr,err := net.ResolveUDPAddr("udp4", "192.168.8.255:8813")
    if err != nil {
        panic(err)
    }

    wg := sync.WaitGroup{}
    wg.Add(1)
    go client_listener(&wg, redisConn)

    for {
        time.Sleep(1/2 * time.Second)
        bcastConn.WriteTo([]byte(load_history(redisConn)), baddr)
        if err != nil {
            panic(err)
        }
    }
    wg.Wait()


}
