package main

import (
	"flag"
	"fmt"
	"github.com/rkjdid/util"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

const Version = "v0"

var (
	cfg *Config
)

var (
	cfgPath  = flag.String("cfg", "", "path to config (defaults to <root>/config.toml)")
	rootPath = flag.String("root", "", "path to goregen's main directory (defaults to executable path)")
	logDir   = flag.String("log", "", "path to logs directory (defaults to <root>/log)")
	version  = flag.Bool("version", false, "print version & exit")
)

func init() {
	flag.Parse()

	// print version & exit
	if *version {
		fmt.Printf("bitbot %s\n", Version)
		os.Exit(0)
	}

	// root directory for goregen
	if *rootPath == "" {
		exe, err := os.Executable()
		if err != nil {
			log.Fatalf("couldn't get path to executable: %s", err)
		}
		*rootPath = filepath.Dir(exe)
	}

	err := os.MkdirAll(*rootPath, 0755)
	if err != nil {
		log.Fatalf("couldn't mkdir root directory \"%s\": %s", *rootPath, err)
	}

	// create log file
	if *logDir == "" {
		*logDir = filepath.Join(*rootPath, "log")
	}
	err = os.MkdirAll(*logDir, 0755)
	if err != nil {
		log.Fatalf("couldn't mkdir log directory \"%s\": %s", *logDir, err)
	}

	logPath := filepath.Join(*logDir, time.Now().Format("2006-01-02_15h04m05.log"))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("couldn't create log file: %s", err)
	}

	// create log link
	link := "bitbot.log"
	logLink := filepath.Join(*rootPath, link)
	_ = os.Remove(logLink)
	err = os.Symlink(logPath, logLink)
	if err != nil {
		err = os.Link(logPath, logLink)
		if err != nil {
			log.Fatalf("couldn't create \"%s\" link: %s", link, err)
		}
	}

	// log to both Stderr & logFile
	log.SetOutput(io.MultiWriter(logFile, os.Stderr))

	// load config
	if *cfgPath == "" {
		*cfgPath = filepath.Join(*rootPath, "config.toml")
	}
	err = util.ReadTomlFile(&cfg, *cfgPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("error reading config \"%s\": %s", *cfgPath, err)
		}
		cfg = &DefaultConfig
		err = util.WriteTomlFile(cfg, *cfgPath)
		if err != nil {
			log.Fatalf("error creating config file \"%s\": %s", *cfgPath, err)
		}
		log.Printf("created new config file \"%s\"", *cfgPath)
	}
	if !cfg.IsValid() {
		log.Fatalf("\"%s\" appears malformed, please fix it or delete it", *cfgPath)
	}

	log.Printf("config file: %s", *cfgPath)
	util.WriteToml(cfg, os.Stderr)
}

func main() {
	log.Println("Press <Ctrl-C> to quit")

	s := Scanner{
		Config: cfg.Scanner,
	}

	go s.Scan()

	trap := make(chan os.Signal)
	signal.Notify(trap, os.Kill, os.Interrupt)
	<-trap
	fmt.Println()
	log.Println("quit received...")

	cleanExit := make(chan struct{})
	go func() {
		s.Stop()
		close(cleanExit)
	}()

	select {
	case <-time.After(time.Second * 10):
		log.Panicln("quit timeout")
	case <-cleanExit:
	}
}
