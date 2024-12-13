package main

import (
    "fmt"
    "net"
    "net/http"
    "os/exec"
    "time"
)

const password = "yourpassword" // 设置你的密码

func addIPToWhitelist(ip string) error {
    cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule", "name=Allow RDP from "+ip, "dir=in", "action=allow", "protocol=TCP", "localport=3389", "remoteip="+ip)
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
    if userPassword != password {
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
        time.Sleep(24 * time.Hour)
        removeIPFromWhitelist(ip)
    }(clientIP)
}

func main() {
    http.HandleFunc("/add_ip", handler)
    fmt.Println("Server started at :8062")
    http.ListenAndServe(":8062", nil)
}
