package main

import (
    "bufio"
    "fmt"
    "github.com/luci/go-render/render"
    "io"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "log"
    "log/syslog"
    "flag"
    "github.com/messyidea/qtunnel/tunnel"
)

type Config struct {
    faddr string
    baddr string
    clientMode bool
    cryptoMethod string
    secret string
}

func waitSignal() {
    var sigChan = make(chan os.Signal, 1)
    signal.Notify(sigChan)
    for sig := range sigChan {
        if sig == syscall.SIGINT || sig == syscall.SIGTERM {
            log.Printf("terminated by signal %v\n", sig)
            return
        } else {
            log.Printf("received signal: %v, ignore\n", sig)
        }
    }
}

func getLines(path string) []string {
    var lines []string
    fi, err := os.Open(path)
    if err != nil {
        log.Fatal("Open file error, err:%v", err)
        return lines
    }
    defer fi.Close()

    br := bufio.NewReader(fi)
    for {
        a, _, c := br.ReadLine()
        if c == io.EOF {
            break
        }
        line := string(a)
        line = strings.TrimSpace(line)
        if len(line) == 0 {
            continue
        }
        lines = append(lines, string(a))
    }
    return lines
}

func getConfigFromFile(conf string) []*Config {
    lines := getLines(conf)
    var result []*Config
    for _, each := range lines {
        if each[0] == '#' {
            continue
        }
        values := strings.Fields(each)
        if len(values) < 5 {
            continue
        }
        var newConfig Config
        newConfig.faddr = values[0]
        newConfig.baddr = values[1]
        newConfig.cryptoMethod = values[2]
        if values[3] == "1" {
            newConfig.clientMode = true
        } else {
            newConfig.clientMode = false
        }
        newConfig.secret = values[4]
        result = append(result, &newConfig)
    }

    fmt.Println("config from file:")
    for _, each := range result {
        fmt.Println(render.Render(each))
    }
    return result
}

func main() {
    var faddr, baddr, cryptoMethod, secret, logTo, conf string
    var clientMode bool
    flag.StringVar(&logTo, "logto", "stdout", "stdout or syslog")
    flag.StringVar(&faddr, "listen", ":9001", "host:port qtunnel listen on")
    flag.StringVar(&baddr, "backend", "127.0.0.1:6400", "host:port of the backend")
    flag.StringVar(&cryptoMethod, "crypto", "rc4", "encryption method")
    flag.StringVar(&secret, "secret", "secret", "password used to encrypt the data")
    flag.BoolVar(&clientMode, "clientmode", false, "if running at client mode")
    flag.StringVar(&conf, "conf", "", "config file")
    flag.Parse()

    log.SetOutput(os.Stdout)
    if logTo == "syslog" {
        w, err := syslog.New(syslog.LOG_INFO, "qtunnel")
        if err != nil {
            log.Fatal(err)
        }
        log.SetOutput(w)
    }

    var configs []*Config
    if conf == "" {
        var newConfig Config
        newConfig.baddr = baddr
        newConfig.faddr = faddr
        newConfig.clientMode = clientMode
        newConfig.cryptoMethod = cryptoMethod
        newConfig.secret = secret
        configs = append(configs, &newConfig)
    } else {
        configs = getConfigFromFile(conf)
    }

    for _, each := range configs {
        t := tunnel.NewTunnel(each.faddr, each.baddr, each.clientMode, each.cryptoMethod, each.secret, 4096)
        go t.Start()
    }

    waitSignal()
}
