package main

import (
	"log"
	"net/http"

	"encoding/json"

	"github.com/maxence-charriere/go-app/v9/pkg/app"

	"image"
	_ "image/png"
	"strings"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/vincent-petithory/dataurl"
)

//build
//$Env:GOARCH="wasm";$Env:GOOS="js";go build -o web/app.wasm

// The main function is the entry point where the app is configured and started.
// It is executed in 2 different environments: A client (the web browser) and a
// server.
func main() {
	// The first thing to do is to associate the hello component with a path.
	//
	// This is done by calling the Route() function,  which tells go-app what
	// component to display for a given path, on both client and server-side.
	app.Route("/", &qreader{})
	// Once the routes set up, the next thing to do is to either launch the app
	// or the server that serves the app.
	//
	// When executed on the client-side, the RunWhenOnBrowser() function
	// launches the app,  starting a loop that listens for app events and
	// executes client instructions. Since it is a blocking call, the code below
	// it will never be executed.
	//
	// On the server-side, RunWhenOnBrowser() does nothing, which allows the
	// writing of server logic without needing precompiling instructions.
	app.RunWhenOnBrowser()

	// Finally, launching the server that serves the app is done by using the Go
	// standard HTTP package.
	//
	// The Handler is an HTTP handler that serves the client and all its
	// required resources to make it work into a web browser. Here it is
	// configured to handle requests with a path that starts with "/".
	http.Handle("/", &app.Handler{
		Author:      "abcdm",
		Name:        "qreader",
		Description: "An Hello World! example",
		Styles: []string{
			"/web/tailwind.css", // Loads hello.css file.
		},
	})

	http.HandleFunc("/decode", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		type Barcode struct {
			Tip    string `json:"tip"`
			Code   string `json:"code"`
			Errmsg string `json:"err"`
		}
		var data string
		var bc = Barcode{Tip: "qrcode", Code: "?", Errmsg: "ok"}
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
			bc = Barcode{Tip: "", Code: "", Errmsg: err.Error()}
		}

		data = r.PostFormValue("data")
		log.Printf("%v\n", data)
		dataURL, err := dataurl.DecodeString(data)
		if err != nil {
			log.Println(err)
			bc = Barcode{Tip: "none", Code: "?", Errmsg: err.Error()}
		} else {
			rdr := strings.NewReader(string(dataURL.Data))
			img, _, _ := image.Decode(rdr)

			// prepare BinaryBitmap
			bmp, _ := gozxing.NewBinaryBitmapFromImage(img)

			// decode image
			qrReader := qrcode.NewQRCodeReader()
			result, err := qrReader.Decode(bmp, nil)
			if err != nil {
				bc = Barcode{Tip: "none", Code: "?", Errmsg: err.Error()}
			} else {
				bc.Code = result.String()
				bc.Tip = "qrcode"
			}
		}
		log.Printf("%v\n", bc)
		rw.WriteHeader(http.StatusCreated)
		enc := json.NewEncoder(rw)
		if err := enc.Encode(bc); err != nil {
			log.Println(err)
		}
		//json.Marshal(bc)
	})

	if err := http.ListenAndServeTLS(":8000", "local.crt", "local.key", nil); err != nil {
		log.Fatal(err)
	}
}
