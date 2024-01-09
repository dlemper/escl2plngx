package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/signintech/gopdf"
)

type ScannerStatus struct {
	XMLName  xml.Name `scan:"ScannerStatus"`
	AdfState string   `scan:"scan:AdfState"`
}

func main() {
	resp, err := http.Get("http://192.168.1.74/eSCL/ScannerStatus")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	byteValue, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	//fmt.Println(string(byteValue))

	var scannerStatus ScannerStatus
	xml.Unmarshal(byteValue, &scannerStatus)
	fmt.Println(scannerStatus.AdfState)

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()
}
