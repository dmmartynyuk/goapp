# goapp
Testing Wasm Technology.
trying to implement BarcodeDetector object in Golang.
Thanks /maxence-charriere/go-app
General idea. 
If the webcam is available, then the barcode is photographed and the drawing is sent for processing. 
In the presence of BarcodeDetection API, we decode. Otherwise, we send the drawing to the server, where we decode the barcode. 
From the server we get a json object with a barcode line.
If no webcam, then we can upload barcode image for detection
