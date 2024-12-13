package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net"
    "net/http"
    "os"
    "os/exec"
)

type Config struct {
    Password string `json:"password"`
    RdpPort  int    `json:"rdp_port"`
}

var config Config

func loadConfig() error {
    file, err := os.Open("config.json")
    if err != nil {
        return err
    }
    defer file.Close()

    decoder := json.NewDecoder(file)
    err = decoder.Decode(&config)
    if err != nil {
        return err
    }

    return nil
}

func addIPToWhitelist(ip string) error {
    cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule", "name=Allow RDP from "+ip, "dir=in", "action=allow", "protocol=TCP", "localport="+fmt.Sprint(config.RdpPort), "remoteip="+ip)
    return cmd.Run()
}

func removeIPFromWhitelist(ip string) error {
    cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name=Allow RDP from "+ip)
    return cmd.Run()
}

func handler(w http.ResponseWriter, r *http.Request) {
    clientIP := r.RemoteAddr
    if ip, _, err := net.SplitHostPort(clientIP); err == nil {
        clientIP = ip
    }

    userPassword := r.URL.Query().Get("password")
    if userPassword != config.Password {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    err := addIPToWhitelist(clientIP)
    if err != nil {
        http.Error(w, "Failed to add IP to whitelist", http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, "IP %s has been added to the whitelist", clientIP)

    // 设置定时任务在5小时后删除IP
    go func(ip string) {
        time.Sleep(5 * time.Hour)
        removeIPFromWhitelist(ip)
    }(clientIP)
}

func main() {
    err := loadConfig()
    if err != nil {
        fmt.Println("Error loading config:", err)
        return
    }

    http.HandleFunc("/add_ip", handler)
    fmt.Println("Server started at :8062")
    http.ListenAndServe(":8062", nil)
}

