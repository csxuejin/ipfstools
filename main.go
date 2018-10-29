package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"
)

type Config struct {
	AddFileWorkerNum    int `json:"add_file_worker_num"`
	PinAddFileWorkerNum int `json:"pin_add_file_worker_num"`
	PinAddWaitTime      int `json:"pin_add_wait_time"`
}

const (
	DEFAULT_FOLDER = "testfiles"
	HASH_FILE      = "filehashes"
)

var (
	config = Config{
		AddFileWorkerNum:    10,
		PinAddFileWorkerNum: 10,
	}

	hashFileAbsPath     string
	currentPath         string
	defaultAddFileWG    sync.WaitGroup
	defaultPinAddFileWG sync.WaitGroup
)

func init() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
		panic("os.GetWd(): " + err.Error())
	}

	configFilePath := path.Join(currentPath, "config.json")
	if _, err := os.Stat(configFilePath); os.IsExist(err) {
		data, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%v): %v\n", configFilePath, err)
			return
		}

		if err := json.Unmarshal(data, &config); err != nil {
			log.Fatalf("json.Unmarshal(): %v\n", err)
			return
		}
	}

	hashFileAbsPath = path.Join(currentPath, HASH_FILE)
	log.Printf("config is %#v\n", config)
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:    "add",
			Aliases: []string{"add"},
			Usage:   "add file or files in the folder",
			Action:  AddFiles,
		},
		{
			Name:    "pinadd",
			Aliases: []string{"pinadd"},
			Usage:   "pin & add files according to filehashes record",
			Action:  PinAddFiles,
		},
		{
			Name:    "pinrm",
			Aliases: []string{"pinrm"},
			Usage:   "pin rm files",
			Action:  PinRmFiles,
		},
		{
			Name:    "rmall",
			Aliases: []string{"rmall"},
			Usage:   "pin rm all files",
			Action:  PinRmAllFiles,
		},
		{
			Name:    "gc",
			Aliases: []string{"gc"},
			Usage:   "ipfs repo gc",
			Action:  GC,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

//////// 'add' operation
func AddFiles(c *cli.Context) error {
	if _, err := os.Create(hashFileAbsPath); err != nil {
		log.Fatalf("create filehashes failed: %v\n")
		return err
	}

	filePath := path.Join(currentPath, DEFAULT_FOLDER)
	fi, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
		return err
	}

	jobs := make(chan string, 200)
	defaultAddFileWG.Add(config.AddFileWorkerNum)
	for w := 0; w < config.AddFileWorkerNum; w++ {
		go WorkerForAdd(jobs)
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			jobs <- path.Join(filePath, f.Name())
		}

	case mode.IsRegular():
		// do file stuff
		jobs <- path.Join(filePath, fi.Name())
	}

	close(jobs)
	defaultAddFileWG.Wait()

	return nil
}

func WorkerForAdd(jobs <-chan string) {
	f, err := os.OpenFile(hashFileAbsPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}

	defer defaultAddFileWG.Done()

	for filePath := range jobs {
		log.Println("filePath is : ", filePath)

		data, err := exec.Command("bash", "-c", "ipfs --api /ip4/127.0.0.1/tcp/9095 add "+filePath).Output()
		if err != nil {
			log.Fatal(err)
		}

		result := string(data)
		if strings.HasPrefix(result, "added") {
			arr := strings.Split(string(data), " ")
			if len(arr) == 3 {
				_, err := f.WriteString(fmt.Sprintf("%v\n", arr[1]))

				if err != nil {
					log.Fatalf("file.WriteString(%v): %v\n", arr[1], err)
				}
			}
		}
	}

}

//////// 'pin add' operation
func PinAddFiles(c *cli.Context) error {
	log.Printf("time before pinadd op: %v\n", time.Now())
	jobs := make(chan string, 200)
	defaultPinAddFileWG.Add(config.PinAddFileWorkerNum)
	for w := 0; w < config.PinAddFileWorkerNum; w++ {
		go WorkerForPinAdd(jobs)
	}

	data, err := ioutil.ReadFile(hashFileAbsPath)
	if err != nil {
		log.Fatalf("ioutil.ReadFile(%v): %v\n", hashFileAbsPath, err)
		return nil
	}

	hashes := strings.Split(string(data), "\n")
	for _, v := range hashes {
		if v != "" {
			jobs <- v
		}
		if config.PinAddWaitTime > 0 {
			log.Printf("Let's sleep %v minutes.\n", config.PinAddWaitTime)
			time.Sleep(time.Minute * time.Duration(config.PinAddWaitTime))
		}
	}
	close(jobs)
	defaultPinAddFileWG.Wait()

	log.Printf("time after pinadd op: %v\n", time.Now())

	return nil
}

func WorkerForPinAdd(jobs <-chan string) {
	defer defaultPinAddFileWG.Done()

	for hash := range jobs {
		log.Println("hash is : ")

		data, err := exec.Command("bash", "-c", "ipfs --api /ip4/127.0.0.1/tcp/9095 pin add "+hash).Output()

		if err != nil {
			log.Fatal(err)
		}

		log.Printf("data is %v\n", string(data))
	}
}

func PinRmFiles(c *cli.Context) error {
	log.Printf("time before pin rm op: %v\n", time.Now())

	data, err := ioutil.ReadFile(hashFileAbsPath)
	if err != nil {
		log.Fatalf("ioutil.ReadFile(%v): %v\n", hashFileAbsPath, err)
		return nil
	}

	hashes := strings.Split(string(data), "\n")
	for _, hash := range hashes {
		if hash != "" {
			log.Printf("ipfs pin rm %v\n", hash)

			data, err := exec.Command("bash", "-c", "ipfs pin rm "+hash).Output()
			if err != nil {
				log.Fatal(err)
			}

			log.Printf("data is %v\n", string(data))
		}
	}

	log.Printf("time after pin rm op: %v\n", time.Now())
	return nil
}

func PinRmAllFiles(c *cli.Context) error {
	data, err := exec.Command("bash", "-c", "ipfs --api /ip4/127.0.0.1/tcp/9095 pin ls --type recursive | cut -d' ' -f1 | xargs -n1 ipfs --api /ip4/127.0.0.1/tcp/9095 pin rm").Output()
	if err != nil {
		log.Fatal(err)
		return err
	}

	log.Printf("result is %v\n", string(data))
	return nil
}

func GC(c *cli.Context) error {
	log.Printf("Time before gc op: %v\n", time.Now())

	data, err := exec.Command("bash", "-c", "ipfs repo gc").Output()
	if err != nil {
		log.Fatalf("ipfs repo gc : %v\n", err)
	}

	log.Printf("result is %v\n", string(data))
	log.Printf("Time after gc op: %v\n", time.Now())

	return nil
}
