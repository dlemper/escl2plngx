package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/robfig/cron/v3"
)

type ScannerStatus struct {
	XMLName  xml.Name `scan:"ScannerStatus"`
	AdfState string   `scan:"scan:AdfState"`
}

func main() {
	scheduler := cron.New()
	entryId, err := scheduler.AddFunc("@every 2s", func() {
		resp, err := http.Get("http://192.168.1.74/eSCL/ScannerStatus")
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		byteValue, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}

		var scannerStatus ScannerStatus
		xml.Unmarshal(byteValue, &scannerStatus)
		//fmt.Println(scannerStatus.AdfState)
		//fmt.Println(resp.StatusCode)

		if scannerStatus.AdfState == "ScannerAdfEmpty" {
			return
		}

		fmt.Println("Scanning...")

		scheduler.Stop()

		//colormode: RGB24, Grayscale8
		scanJobBody := strings.NewReader("<?xml version=\"1.0\" encoding=\"UTF-8\"?><scan:ScanSettings xmlns:scan=\"http://schemas.hp.com/imaging/escl/2011/05/03\" xmlns:pwg=\"http://www.pwg.org/schemas/2010/12/sm\"><pwg:Version>2.0</pwg:Version><scan:Intent>TextAndGraphic</scan:Intent><pwg:ScanRegions pwg:MustHonor=\"true\"><pwg:ScanRegion><pwg:ContentRegionUnits>escl:ThreeHundredthsOfInches</pwg:ContentRegionUnits><pwg:Width>2480</pwg:Width><pwg:Height>3508</pwg:Height><pwg:XOffset>0</pwg:XOffset><pwg:YOffset>0</pwg:YOffset></pwg:ScanRegion></pwg:ScanRegions><pwg:DocumentFormat>image/jpeg</pwg:DocumentFormat><pwg:InputSource>Feeder</pwg:InputSource><scan:ColorMode>RGB24</scan:ColorMode><scan:XResolution>300</scan:XResolution><scan:YResolution>300</scan:YResolution><scan:Duplex>false</scan:Duplex></scan:ScanSettings>")
		resp, err = http.Post("http://192.168.1.74/eSCL/ScanJobs", "application/xml", scanJobBody)
		// POST http://${scannerIp}/eSCL/ScanJobs => Location header in response + NextDocument
		// GET http://${scannerIp}${uuid}/NextDocument for each page until body is empty
		// POST   multipart body with "document" field
		loc := resp.Header.Get("Location") + "/NextDocument"

		pdf := fpdf.New("P", "mm", "A4", "")

		pageCtr := 0
		for {
			resp, err = http.Get(loc)

			if resp.StatusCode == 404 {
				break
			}

			pdf.AddPage()
			pageCtr++
			//imgHolder, _ := gopdf.ImageHolderByReader(resp.Body)
			//pdf.ImageByHolder(imgHolder, 0, 0, gopdf.PageSizeA4)
			_ = pdf.RegisterImageOptionsReader("page"+strconv.Itoa(pageCtr), fpdf.ImageOptions{ImageType: "jpg", ReadDpi: true}, resp.Body)
			pdf.ImageOptions("page"+strconv.Itoa(pageCtr), 0, 0, -300, -300, false, fpdf.ImageOptions{
				ImageType:             "JPG",
				ReadDpi:               true,
				AllowNegativePosition: false,
			}, 0, "")
		}

		//fmt.Println(pdf.GetNumberOfPages())

		if pageCtr > 0 {
			buf := new(bytes.Buffer)
			bw := multipart.NewWriter(buf)
			fw, _ := bw.CreateFormFile("document", time.Now().Format("2006-01-02-15-04-05"))
			//pdf.WriteTo(fw)
			//pdf.WritePdf("./" + time.Now().Format("2006-01-02-15-04-05") + ".pdf")
			pdf.Output(fw)
			pdf.Close()
			bw.Close()

			req, _ := http.NewRequest("POST", "http://nas:8000/api/documents/post_document/", buf)
			req.Header.Add("Authorization", "Token ba745bad3030fef2f6836ea3e65d259a230fd5cb")
			req.Header.Add("Content-Type", bw.FormDataContentType())
			client := &http.Client{}
			client.Do(req)

			//fmt.Println("post", err, res)
		}

		scheduler.Start()
	})

	fmt.Println(entryId, err)

	scheduler.Start()

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel
}
