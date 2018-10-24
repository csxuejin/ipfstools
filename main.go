package main

import (
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

const (
	DEFAULT_FOLDER = "testfiles"
	FILE_HASH      = "filehashes"
)

var (
	fileHashPath string
)

func init() {
	filePath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
		panic("os.GetWd(): " + err.Error())
	}

	fileHashPath = path.Join(filePath, FILE_HASH)
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

var (
	addFileWG        sync.WaitGroup
	addFileWorkerNum = 10
)

func AddFiles(c *cli.Context) error {
	filePath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
		return err
	}

	hashFilePath := path.Join(filePath, FILE_HASH)
	if _, err := os.Create(hashFilePath); err != nil {
		log.Fatalf("create filehashes failed: %v\n")
		return err
	}

	switch c.NArg() {
	case 1:
		filePath = c.Args()[0]

	case 0:
		// DO Nothing
		filePath = path.Join(filePath, DEFAULT_FOLDER)

	default:
		log.Fatal("Wrong Arguments.")
		return nil
	}

	fi, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
		return err
	}

	jobs := make(chan string, 200)
	addFileWG.Add(addFileWorkerNum)
	for w := 0; w < addFileWorkerNum; w++ {
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
	addFileWG.Wait()

	return nil
}

func WorkerForAdd(jobs <-chan string) {
	f, err := os.OpenFile(fileHashPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}

	defer addFileWG.Done()

	for filePath := range jobs {
		fmt.Println("filePath is : ", filePath)

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

var (
	pinAddFileWG        sync.WaitGroup
	pinAddFileWorkerNum = 10
)

func PinAddFiles(c *cli.Context) error {
	fmt.Printf("time before pinadd op: %v\n", time.Now())
	jobs := make(chan string, 200)
	pinAddFileWG.Add(pinAddFileWorkerNum)
	for w := 0; w < pinAddFileWorkerNum; w++ {
		go WorkerForPinAdd(jobs)
	}

	data, err := ioutil.ReadFile(fileHashPath)
	if err != nil {
		log.Fatalf("ioutil.ReadFile(%v): %v\n", fileHashPath, err)
		return nil
	}

	hashes := strings.Split(string(data), "\n")
	for _, v := range hashes {
		if v != "" {
			jobs <- v
		}
	}
	close(jobs)
	pinAddFileWG.Wait()

	fmt.Printf("time after pinadd op: %v\n", time.Now())

	return nil
}

func WorkerForPinAdd(jobs <-chan string) {
	defer pinAddFileWG.Done()

	for hash := range jobs {
		fmt.Println("hash is : ", hash)

		data, err := exec.Command("bash", "-c", "ipfs --api /ip4/127.0.0.1/tcp/9095 pin add "+hash).Output()

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("data is %v\n", string(data))
	}
}

func PinRmFiles(c *cli.Context) error {
	fmt.Printf("time before pin rm op: %v\n", time.Now())

	data, err := ioutil.ReadFile(fileHashPath)
	if err != nil {
		log.Fatalf("ioutil.ReadFile(%v): %v\n", fileHashPath, err)
		return nil
	}

	hashes := strings.Split(string(data), "\n")
	for _, hash := range hashes {
		if hash != "" {
			fmt.Printf("ipfs pin rm %v\n", hash)

			data, err := exec.Command("bash", "-c", "ipfs pin rm "+hash).Output()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("data is %v\n", string(data))
		}
	}

	fmt.Printf("time after pin rm op: %v\n", time.Now())

	return nil
}

func GC(c *cli.Context) error {
	fmt.Printf("Time before gc op: %v\n", time.Now())

	data, err := exec.Command("bash", "-c", "ipfs repo gc").Output()
	if err != nil {
		log.Fatalf("ipfs repo gc : %v\n", err)
	}

	fmt.Printf("result is %v\n", string(data))

	fmt.Printf("Time after gc op: %v\n", time.Now())
	return nil
}
