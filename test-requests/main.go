package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	uuid "github.com/satori/go.uuid"
)

const tabs = "\t"

var (
	addr        = flag.String("addr", "localhost:7778", "address")
	num         = flag.Int("count", 1, "number of requests")
	enableHTTPS = flag.Bool("https", false, "enable https")
)

func init() {
	flag.Parse()
}

var errWrongStatusCode = errors.New("wrong status code")

func process(method, url string, body io.Reader, code int) error {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("path: [%s], err :[%s]", url, err.Error())
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("path: [%s], err: [%s]", url, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != code {
		return fmt.Errorf("path [%s], err [status code: got %d, expected %d]", url, resp.StatusCode, code)
	}

	return nil
}

func runTest(wg *sync.WaitGroup) {
	defer wg.Done()

	//****************************
	// GET /server/config
	//****************************

	path := "/server/config"

	err := process("GET", *addr+path, nil, 200)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%s  %s: [200]\n", tabs, path)

	u, err := uuid.NewV4()
	body := strings.NewReader(`{
			"userID": "` + u.String() + `",
			"deviceID":	"Test phone 3000",
			"deviceType": 2,
			"pushToken": "nop"
		}`)

	path = "/auth"
	req, err := http.NewRequest("POST", *addr+path, body)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()

	type WithToken struct {
		Token string `json:"token"`
	}

	var s = WithToken{}
	err = json.NewDecoder(resp.Body).Decode(&s)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}

	token := string(s.Token)
	log.Printf("%s  %s: [200]\n", tabs, path)

	//****************************
	// POST /api/v1/wallet
	//****************************

	path = "/api/v1/wallet"

	body = strings.NewReader(`{
			   "currencyID": 0,
			   "address":	"nLsCSu8bQUsfJHCBQaVgXS5vmVQcSkQNmq",
			   "addressIndex": 2,
			   "walletIndex": 3,
			   "walletName": "sobaka3"
		}`)

	req, err = http.NewRequest("POST", *addr+path, body)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		log.Fatalf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}
	log.Printf("%s  %s: [200]\n", tabs, path)

	//****************************
	// POST /api/v1/address
	//****************************

	path = "/api/v1/address"

	body = strings.NewReader(`{
			"walletIndex": 3,
			"address": "new address",
			"addressIndex": 8,
			"currencyID": 0
		}`)

	req, err = http.NewRequest("POST", *addr+path, body)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		log.Panicf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}

	log.Printf("%s  %s: [200]\n", tabs, path)

	//****************************
	// GET /api/v1/transaction/feerate/0
	//****************************

	path = "/api/v1/transaction/feerate/0"

	req, err = http.NewRequest("GET", *addr+path, nil)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}

	log.Printf("%s  %s: [200]\n", tabs, path)

	//****************************
	// GET /api/v1/outputs/spendable/0/n34J7cyue3NjttDeCdAn7XcPnVKNb2CxAg
	//****************************

	path = "/api/v1/outputs/spendable/0/n34J7cyue3NjttDeCdAn7XcPnVKNb2CxAg"

	req, err = http.NewRequest("GET", *addr+path, nil)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}
	log.Printf("%s  %s: [200]\n", tabs, path)

	//****************************
	// POST /api/v1/transaction/send/0
	//****************************

	path = "/api/v1/transaction/send/0"

	body = strings.NewReader(`{
			"transaction":"test"
		}`)

	req, err = http.NewRequest("POST", *addr+path, body)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		log.Fatalf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}

	log.Printf("%s  %s: [200]\n", tabs, path)

	//****************************
	// GET /api/v1/wallet/3/verbose/0
	//****************************

	path = "/api/v1/wallet/3/verbose/0"

	req, err = http.NewRequest("GET", *addr+path, nil)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}

	log.Printf("%s  %s: [200]\n", tabs, path)

	//****************************
	// GET /api/v1/wallets/verbose
	//****************************

	path = "/api/v1/wallets/verbose"

	req, err = http.NewRequest("GET", *addr+path, nil)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}

	log.Printf("%s  %s: [200]\n", tabs, path)

	//****************************
	// POST /api/v1/wallet/name/0
	//****************************

	path = "/api/v1/wallet/name/0/3"

	body = strings.NewReader(`{
			"walletname":"kek"
		}`)

	req, err = http.NewRequest("POST", *addr+path, body)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}

	log.Printf("%s  %s: [200]\n", tabs, path)

	//************************
	// delete /v1/wallet/0/3
	//************************

	path = "/api/v1/wallet/0/3"

	req, err = http.NewRequest("DELETE", *addr+path, nil)
	if err != nil {
		log.Fatalf("path: [%s], err: [%s]\n", path, err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("path: [%s] err: [%s]\n", path, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Panicf("path=%s, err= wrong status code: %d", path, resp.StatusCode)
	}
	log.Printf("%s  %s: [200]\n", tabs, path)

}

func main() {
	if *enableHTTPS {
		*addr = "https://" + *addr
	} else {
		*addr = "http://" + *addr
	}

	var wg sync.WaitGroup
	wg.Add(*num)
	for i := 0; i < *num; i++ {
		go func(wg *sync.WaitGroup) {
			runTest(wg)
		}(&wg)
	}
	wg.Wait()
	log.Println("\t  Done")
}
