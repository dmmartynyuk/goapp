package main

import (
	"errors"
	//"syscall/js"
	"time"

	"encoding/json"
	"net/http"
	"net/url"

	"github.com/maxence-charriere/go-app/v9/pkg/app"
)

type qreader struct {
	app.Compo
	Val               string
	isWebCam          string
	isBarcodeDetector bool
	height            int
	width             int
	errmsg            string
	mediadevices      []string
	stopscan          bool
	shouldFaceUser    bool
	canFaceMode       bool
}

/*
			<div id=camera class=Camera>
                <canvas class=Camera-display></canvas>
                <div class="CameraRealtime hidden">
                    <video class=Camera-video muted autoplay playsinline></video>
                    <div class=Camera-instructions>Point at a QR code</div>
                    <div class="Camera-toggle wsk-button wsk-button--fab">
                        <input type=checkbox id=checkbox-1 class=Camera-toggle-input>
                        <label for=checkbox-1>
                            <img class=front src=/images/ic_camera_front_24px.svg>
                            <img class=rear src=/images/ic_camera_rear_24px.svg>
                        </label>
                    </div>
                </div>
                <div class=Camera-overlay></div>
                <div id=camera-fallback class="CameraFallback hidden">
                    <form method=post action=/scan-image class=CameraFallback-form>
                        <input type=file accept=image/* capture=camera id=qrcode-input name=qrcode class=CameraFallback-input>
                        <label for=qrcode-input>
                            <img src=/images/ic_file_upload_24px.svg>
                        </label>
                        <img class=CameraFallback-image>
                    </form>
                </div>
            </div>
*/

func (h *qreader) Render() app.UI {
	return app.Div().ID("camera").Class("camera").Body(
		app.Canvas().ID("ctx").Style("display", "none").Width(h.width).Height(h.height),
		app.Div().Class("CameraRealtime").Body(
			app.VideoExt().Playsinline(true).ID("qrvcam").AutoPlay(true).Muted(true).Class("camera-video").Style("display", h.isWebCam), //Height(0).Width(0)  Style("transform", "translate(-50%, -50%) scale(5.6625)")
		),
		app.Button().ID("btnscan").Class("btnscan").
			Text("123").
			OnClick(h.ScanCode),
		app.Button().ID("btntoggile").Class("btntoggile").OnClick(h.CameraToggile),
		app.Img().ID("loadedimg").Src("/web/images/barcode.png").Style("display", "none").OnLoad(h.OnLoadImage),
		app.Div().Class("camera-overlay").Body(),
		app.Div().Class("bcdetected").Body(
			app.Label().For("qrcode-input").Body(
				app.Img().Src("/web/images/barcode.svg").Height(24).Width(24),
			),
			app.Input().
				Type("text").
				ID("qrcode-input").
				Value(h.Val).
				Placeholder("code, please").
				AutoFocus(false).
				OnChange(h.ValueTo(&h.Val)),
		),
		app.Div().ID("CamFall").Body(
			app.Form().ID("getfile").Method("post").Action("/scan-image").Body(
				//app.Raw(`<input type="file" id="qrcode-img" placeholder="select image, please" class="CameraFallback-input" accept="image/*" capture="camera" value="" autoFocus="true" />`),
				app.InputExt().Capture("camera").
					Type("file").
					ID("qrcode-img").
					Class("CameraFallback-input").
					Accept("image/*").
					Placeholder("select image, please").
					AutoFocus(true).
					OnChange(h.OnLoadFiles),
			),
		),
		app.If(!h.isBarcodeDetector,
			app.H5().Text("Barcode not detected!"),
		),
		app.If(h.isWebCam == "none",
			app.H5().Text("WEBCAM not detected!"),
		),
	)
}

func (h *qreader) CameraToggile(ctx app.Context, e app.Event) {
	if len(h.mediadevices) == 0 {
		return
	}
	video := app.Window().GetElementByID("qrvcam")
	video.Call("pause")
	stream := video.Get("srcObject")
	tracks := stream.Call("getTracks")
	for i := 0; i < tracks.Length(); i++ {
		tracks.Index(i).Call("stop")
	}
	video.Set("srcObject", app.Null())
	h.shouldFaceUser = !h.shouldFaceUser
	h.CaptureCam(ctx)
}

func (h *qreader) searchbarcode(ctx app.Context) {
	canvas := app.Window().GetElementByID("ctx")
	context := canvas.Call("getContext", "2d")
	video := app.Window().GetElementByID("qrvcam")
	app.Window().GetElementByID("qrcode-input").Set("value", h.Val)
	for !h.stopscan || len(h.Val) == 0 {
		ctx.Async(func() {
			context.Call("drawImage", video, 0, 0, h.width, h.height)
			//surl := canvas.Call("toDataURL", "image/png")
			rawdata := context.Call("getImageData", 0, 0, h.width, h.height)
			//data := rawdata.Get("data")
			ret, err := h.BarcodeDerect(ctx, rawdata)
			if err == nil && len(ret) > 0 {
				h.Val = ret
			} else {
				app.Window().GetElementByID("qrcode-input").Set("value", err.Error())
			}
		})
		app.Window().GetElementByID("qrcode-input").Set("value", h.Val)
		time.Sleep(250 * time.Millisecond)
	}

}

func (h *qreader) ScanCode(ctx app.Context, e app.Event) {
	canvas := app.Window().GetElementByID("ctx")
	context := canvas.Call("getContext", "2d")
	video := app.Window().GetElementByID("qrvcam")
	context.Call("drawImage", video, 0, 0, h.width, h.height)

	surl := canvas.Call("toDataURL", "image/png")
	//canvas.toBlob(callback, mimeType, qualityArgument);
	//app.Window().GetElementByID("loadedimg").Set("src", surl)
	h.decode(ctx, surl.String())
	if len(h.mediadevices) > 0 {
		app.Logf(" mediaDevice id= %v", h.mediadevices)
	} else {
		app.Log("no founded id mediadevices")
	}
}

func (h *qreader) decode(ctx app.Context, simg string) {
	ctx.Async(func() {
		http.DefaultClient.Timeout = 10 * time.Second
		sdata := url.Values{
			"data": {simg},
		}
		r, err := http.DefaultClient.PostForm("/decode", sdata)
		if err != nil {
			app.Log(err)
			return
		}
		defer r.Body.Close()

		//b, err := io.ReadAll(r.Body)
		//if err != nil {
		//	app.Log(err)
		//	return
		//}

		type Barcode struct {
			Tip    string `json:"tip"`
			Code   string `json:"code"`
			Errmsg string `json:"err"`
		}
		var target Barcode
		json.NewDecoder(r.Body).Decode(&target)
		if len(target.Code) > 0 {
			app.Window().GetElementByID("qrcode-input").Set("value", target.Code)
			h.Val = target.Code
			h.errmsg = target.Errmsg

		} else {
			app.Window().GetElementByID("qrcode-input").Set("value", target.Errmsg)
			h.Val = target.Code
			h.errmsg = target.Errmsg
		}
		//app.Logf("request response: %s", b)
	})
}

func (h *qreader) OnLoadImage(ctx app.Context, e app.Event) {
	src := ctx.JSSrc().Get("src")
	app.Window().Get("URL").Call("revokeObjectURL", src)
	img := app.Window().GetElementByID("loadedimg")
	//url=img.Get("src")
	canvas := app.Window().GetElementByID("ctx")
	context := canvas.Call("getContext", "2d")
	context.Call("drawImage", img, 0, 0, h.width, h.height)
	surl := canvas.Call("toDataURL", "image/png")
	if h.isWebCam == "none" {
		h.decode(ctx, surl.String())
	}
}

func (h *qreader) OnLoadFiles(ctx app.Context, e app.Event) {
	imgfl := ctx.JSSrc().Get("files").Index(0)
	surl := app.Window().Get("URL").Call("createObjectURL", imgfl)
	img := app.Window().GetElementByID("loadedimg")
	img.Set("src", surl)
	//readr := app.Window().Get("FileReader").New()
	//readr.Call("readAsBinaryString",imgfl)
	h.Update()
}

func (h *qreader) OnNav(ctx app.Context) {
	if h.isWebCam == "none" {
		app.Window().GetElementByID("ctx").Set("style", "z-index:1000;display:block;")
		app.Window().GetElementByID("qrvcam").Set("style", "display:none")
		app.Window().GetElementByID("btnscan").Set("style", "display:none")
	} else {
		app.Window().GetElementByID("CamFall").Set("style", "display:none")
		app.Window().GetElementByID("getfile").Set("style", "display:none")
	}
	if h.isBarcodeDetector {
		app.Window().GetElementByID("btnscan").Set("style", "display:none")
		ctx.Async(func() {
			h.searchbarcode(ctx)
		})
	}
	if !h.canFaceMode {
		app.Window().GetElementByID("btntoggile").Set("style", "display:none")
	}
}

func (h *qreader) CaptureCam(ctx app.Context) {
	var err error
	succCh := make(chan struct{})
	errCh := make(chan error)
	success := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		go func() {
			app.Window().Set("qstream", args[0])
			video := app.Window().GetElementByID("qrvcam")
			video.Set("srcObject", args[0])
			//ctx.JSSrc.Set("srcObject", args[0])
			video.Call("play")
			succCh <- struct{}{}
		}()
		return nil
	})

	failure := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		go func() {
			err = errors.New("failed initialising the camera")
			errCh <- err
		}()
		return nil
	})
	facinMode := app.Window().Get("Object").New()
	if h.shouldFaceUser {
		facinMode.Set("facingMode", "user")
	} else {
		facinMode.Set("facingMode", "environment")
	}
	opts := app.Window().Get("Object").New()
	opts.Set("video", facinMode)

	mediaDevices := app.Window().Get("navigator").Get("mediaDevices")
	if mediaDevices.IsUndefined() {
		err = errors.New("the camera is undefined")
		errCh <- err
	} else {
		promise := mediaDevices.Call("getUserMedia", opts)
		promise.Call("then", success, failure)
	}
	select {
	case <-succCh:
		//video :=   app.Window().GetElementByID("qrvcam")
		h.isWebCam = "inherit"
		go h.getMediaDevices()
		return
	case <-errCh:
		h.isWebCam = "none"
		return
	}
}

func (h *qreader) OnMount(ctx app.Context) {
	var err error
	succCh := make(chan struct{})
	errCh := make(chan error)

	h.isWebCam = "none"
	h.isBarcodeDetector = false
	h.stopscan = false

	h.height = 240
	h.width = 320

	//ctx.ObserveState("barcode").Value(&h.Val)

	h.isBarcodeDetector = !app.Window().Get("BarcodeDetector").IsUndefined()

	//video :=   app.Window().GetElementByID("qrvcam")

	success := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		go func() {
			app.Window().Set("qstream", args[0])
			video := app.Window().GetElementByID("qrvcam")
			video.Set("srcObject", args[0])
			//ctx.JSSrc.Set("srcObject", args[0])
			video.Call("play")
			succCh <- struct{}{}
		}()
		return nil
	})

	failure := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		go func() {
			err = errors.New("failed initialising the camera")
			errCh <- err
		}()
		return nil
	})

	//opts:=ctx.JSSrc.Get("Object").New()
	opts := app.Window().Get("Object").New()

	videoSize := app.Window().Get("Object").New()
	videoSize.Set("width", h.width)
	videoSize.Set("height", h.height)
	videoSize.Set("aspectRatio", 1.777777778)

	opts.Set("video", videoSize)
	opts.Set("audio", false)

	mediaDevices := app.Window().Get("navigator").Get("mediaDevices")
	if mediaDevices.IsUndefined() {
		err = errors.New("the camera is undefined")
		errCh <- err
	} else {

		// check whether we can use facingMode
		supports := mediaDevices.Call("getSupportedConstraints")
		fm := supports.Get("facingMode")
		h.canFaceMode = fm.Bool()

		promise := mediaDevices.Call("getUserMedia", opts)
		promise.Call("then", success, failure)
	}
	select {
	case <-succCh:
		//video :=   app.Window().GetElementByID("qrvcam")
		h.isWebCam = "inherit"
		go h.getMediaDevices()
		return
	case <-errCh:
		h.isWebCam = "none"
		return
	}
	//deviceId (идентификатор устройства). Его значение может быть получено из  метода mediaDevices.enumerateDevices(), возвращающего список, имеющихся на машине устройств,

}

func (h *qreader) getMediaDevices() {
	if h.isWebCam == "none" {
		return
	}
	mediaDevices := app.Window().Get("navigator").Get("mediaDevices")
	if mediaDevices.IsUndefined() {
		return
	}
	succCh := make(chan struct{})
	errCh := make(chan error)
	success := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		go func() {
			sourses := args[0]
			for i := 0; i < sourses.Length(); i++ {
				if sourses.Index(i).Get("kind").String() == "videoinput" {
					h.mediadevices = append(h.mediadevices, sourses.Index(i).Get("deviceId").String())
				}
			}
			succCh <- struct{}{}
		}()
		return nil
	})

	failure := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		go func() {
			err := errors.New("No ennumerated devises")
			errCh <- err
		}()
		return nil
	})

	promise := mediaDevices.Call("enumerateDevices")
	promise.Call("then", success, failure)
	select {
	case <-succCh:
		h.isWebCam = "inherit"
		return
	case <-errCh:
		h.isWebCam = "none"
		return
	}
}

func (h *qreader) BarcodeDerect(ctx app.Context, img app.Value) (string, error) {
	//var barcodeDetector = new BarcodeDetector({formats: ['code_39', 'codabar', 'ean_13']});
	//BarcodeDetector.getSupportedFormats()
	//.then(supportedFormats => {
	// supportedFormats.forEach(format => console.log(format));
	//});
	// barcodeDetector.detect(imageEl)
	// .then(barcodes => {
	//   barcodes.forEach(barcode => console.log(barcode.rawData));
	// })
	// .catch(err => {
	//   console.log(err);
	// })
	var ret string
	//opts := app.Window().Get("Object").New()
	//opts.Set("formats", "['qr_code','ean_8','code_39', 'codabar', 'ean_13']")
	barcodeDetector := app.Window().Get("BarcodeDetector").New()
	// barcodeDetector.Call("getSupportedFormats").Call("then",app.FuncOf(func(this app.Value, args []app.Value) interface{} {

	// })
	succCh := make(chan string)
	errCh := make(chan error)
	success := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		go func() {
			barcodes := args[0]
			// for i:=0; i<barcodes.Length();i++{
			// 	succCh <- barcodes.Index(i).Get("rawData").String()
			// }
			if barcodes.Length() > 0 {
				succCh <- barcodes.Index(0).Get("rawData").String()
			} else {
				err := errors.New("x barcode")
				errCh <- err
			}
		}()
		return nil
	})

	failure := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		go func() {
			err := errors.New("No barcode detected")
			errCh <- err
		}()
		return nil
	})

	canvas := app.Window().GetElementByID("ctx")
	context := canvas.Call("getContext", "2d")
	rawdata := context.Call("getImageData", 0, 0, h.width, h.height)
	//data := rawdata.Get("data").String()
	//app.CopyBytesToJS()

	promise := barcodeDetector.Call("detect", rawdata)
	promise.Call("then", success, failure)
	select {
	case ret = <-succCh:
		return ret, nil
	case err := <-errCh:
		return err.Error(), err
	}

}
