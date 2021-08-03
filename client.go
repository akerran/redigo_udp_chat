package main

import (
    "net"
    "fmt"
    "math/rand"
    "bufio"
    "os"
    "sync"
    "time"
    "strconv"
)

func server_listener(wg *sync.WaitGroup) {
    listenAddr,err := net.ResolveUDPAddr("udp4", ":8813")
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

        fmt.Printf("%s\n", buf[:n])
        time.Sleep(1 * time.Second)
    }
    wg.Done()
}

func main() {
    fmt.Println("Enter your name: ")
    var username string
    fmt.Scanln(&username)


    pc, err := net.ListenPacket("udp4", ":7713")
    if err != nil {
        panic(err)
    }
    defer pc.Close()

    addr,err := net.ResolveUDPAddr("udp4", "192.168.8.247:7713")
    if err != nil {
        panic(err)
    }

    wg := sync.WaitGroup{}
    wg.Add(1)
    go server_listener(&wg)

    reader := bufio.NewReader(os.Stdin)
    rand.Seed(time.Now().UTC().UnixNano())

    for {
        msg, _, _ := reader.ReadLine()
        if err != nil {
            panic(err)
        }
        msgid := rand.Intn(1000000)
        dbrecord := strconv.Itoa(msgid)+" "+username+": "+string(msg)
        pc.WriteTo([]byte(dbrecord), addr)
    }
    wg.Wait()

}

