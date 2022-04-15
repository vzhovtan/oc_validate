package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"github.com/karimra/gnmic/api"
	"github.com/karimra/gnmic/target"
	"google.golang.org/protobuf/encoding/prototext"
)

type TestData struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Encoding string `json:"encoding"`
	SubMode  string `json:"subscriptionMode"`
	Interval string `json:"sampleInterval"`
}

func main() {
	targetPasswd := flag.String("p", "", "a password to connect the the target")
	targetAddr := flag.String("a", "", "IP address and port for the target")
	targetUname := flag.String("u", "admin", "username to connect to to target")
	flag.Parse()

	var tData []TestData
	wg := &sync.WaitGroup{}
	chGet := make(chan string)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a target
	tg, err := api.NewTarget(
		api.Address(*targetAddr),
		api.Username(*targetUname),
		api.Password(*targetPasswd),
		api.SkipVerify(true),
	)
	checkErr(err)

	// create a gNMI client
	err = tg.CreateGNMIClient(ctx)
	checkErr(err)
	defer tg.Close()

	// read file with test cases
	f, err := ioutil.ReadFile("test_cases.json")
	checkErr(err)
	json.Unmarshal(f, &tData)

	// Loop over all test cases, create a request for each test case and send it to the target
	for _, tCase := range tData {
		switch tCase.Method {
		case "get":
			wg.Add(1)
			go getMode(ctx, wg, tg, tCase.Path, tCase.Encoding, chGet)
		case "set":
			fmt.Printf("gNMI set method %+v, %+v, %+v, %+v\n", tCase.Path, tCase.Encoding, tCase.SubMode, tCase.Interval)
		case "subscribe":
			fmt.Printf("gNMI subscribe method %+v, %+v, %+v, %+v\n", tCase.Path, tCase.Encoding, tCase.SubMode, tCase.Interval)
		default:
			log.Fatal("Bad gNMI method")
		}
	}
	go  func() {
		wg.Wait()
		close(chGet)
	}()
	for item := range chGet{
		fmt.Println(item)
	}
}

func getMode(ctx context.Context, wg *sync.WaitGroup, tg *target.Target, path, encoding string, ch chan string) {
	defer wg.Done()
	getReq, err := api.NewGetRequest(api.Encoding(encoding), api.Path(path))
	checkErr(err)
	getResp, err := tg.Get(ctx, getReq)
	checkErr(err)
	ch <- fmt.Sprintf("%v\n",prototext.Format(getResp))
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
