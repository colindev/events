package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type Line struct {
	raw    string
	prefix string
	exists bool
	next   *Line
}

func (l *Line) String() string {

	ret := fmt.Sprintf("%s %s", l.prefix, l.raw)
	if l.exists {
		return fmt.Sprintf("\033[32m%s\033[m", ret)
	}

	return fmt.Sprintf("\033[31m%s\033[m", ret)
}

func main() {

	var (
		launcherFile string
		listenerFile string
	)

	flag.StringVar(&launcherFile, "launcher", "./launcher.log", "launcher log file")
	flag.StringVar(&listenerFile, "listener", "./listener.log", "listener log file")
	flag.Parse()

	fLauncher, err := os.Open(launcherFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fLauncher.Close()

	fListener, err := os.Open(listenerFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fListener.Close()

	cache := map[string]*Line{}

	var startLine, endLine, currentLine *Line
	bufLauncher := bufio.NewReader(fLauncher)
	cntLauncher := 0
	for {
		line, _, err := bufLauncher.ReadLine()
		if err != nil {
			break
		}

		s := strings.TrimSpace(string(line))
		if s == "" {
			continue
		}

		cntLauncher++
		L := &Line{
			raw: s,
		}
		if startLine == nil {
			startLine = L
		}
		endLine = L
		if currentLine != nil {
			currentLine.next = L
		}
		currentLine = L

		cache[L.raw] = L
	}

	notInFire := []string{}
	bufListener := bufio.NewReader(fListener)
	cntListener := 0
	var startListen, endListen string
	statusPrefix := " "
	for {
		line, _, err := bufListener.ReadLine()
		if err != nil {
			break
		}
		if len(line) > 0 && line[0] == '=' {
			switch strings.TrimSpace(string(line[1:])) {
			case " ":
				break
			case "crash":
				statusPrefix = "x"
			case "ready":
				statusPrefix = "o"
			case "connected":
				statusPrefix = "|"
			case "disconnected":
				statusPrefix = "-"
			default:
				statusPrefix = "?"
			}
			continue
		}

		cntListener++

		s := strings.TrimSpace(string(line))
		if s == "" {
			continue
		}

		if startListen == "" {
			startListen = s
		}
		endListen = s

		if l, exists := cache[s]; !exists {
			notInFire = append(notInFire, s)
		} else {
			l.exists = true
			l.prefix = statusPrefix
		}
	}

	fmt.Printf("[fire]: %d \033[32mstart:\033[m %s \033[32mend:\033[m %s\n", cntLauncher, startLine, endLine)
	fmt.Printf("[rece]: %d \033[32mstart:\033[m %s \033[32mend:\033[m %s\n", cntListener, startListen, endListen)
	cntMiss := 0
	L := startLine
	for {
		if L == nil {
			break
		}
		fmt.Println(L.String())
		if !L.exists {
			cntMiss++
		}
		L = L.next
	}
	fmt.Println("miss: ", cntMiss)

	fmt.Println("not in fire:")
	for _, s := range notInFire {
		fmt.Println(s)
	}
}
