// What it does:
//
// This example detects motion using a delta threshold from the first frame,
// and then finds contours to determine where the object is located.
//
// Very loosely based on Adrian Rosebrock code located at:
// http://www.pyimagesearch.com/2015/06/01/home-surveillance-and-motion-detection-with-the-raspberry-pi-python-and-opencv/
//
// How to run:
//
// 		go run ./cmd/motion-detect/main.go 0
//

package main

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strconv"
	"time"

	"gocv.io/x/gocv"
)

const MinimumArea = 3000

func main() {
	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\tmotion-detect [camera ID]")
		return
	}

	// parse args
	deviceID := os.Args[1]

	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Printf("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	window := gocv.NewWindow("Motion Window")
	defer window.Close()

	img := gocv.NewMat()
	defer img.Close()

	imgDelta := gocv.NewMat()
	defer imgDelta.Close()

	imgThresh := gocv.NewMat()
	defer imgThresh.Close()

	mog2 := gocv.NewBackgroundSubtractorMOG2()
	defer mog2.Close()

	status := "Ready"

	dance := 0
	//Amount of time that the system should wait (seconds) before decrementing the dance score
	decrementDuration := time.Duration(2 * 1000000000)

	//Maybe we should limit how often it increments too?
	incrementDuration := time.Duration(1 * 1000000000)

	//Record the time that the system last decremented the dance score.
	timeOfLastDecrement := time.Now()
	timeOfLastIncrement := time.Now()

	fmt.Printf("Start reading device: %v\n", deviceID)
	for {
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		status = "Ready"
		statusColor := color.RGBA{0, 255, 0, 0}

		// first phase of cleaning up image, obtain foreground only
		mog2.Apply(img, &imgDelta)

		// remaining cleanup of the image to use for finding contours.
		// first use threshold
		gocv.Threshold(imgDelta, &imgThresh, 25, 255, gocv.ThresholdBinary)

		// then dilate
		kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
		defer kernel.Close()
		gocv.Dilate(imgThresh, &imgThresh, kernel)

		// now find contours
		contours := gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
		for i := 0; i < contours.Size(); i++ {
			area := gocv.ContourArea(contours.At(i))
			if area < MinimumArea {
				//I want it to stop at 0, but for debugging purposes I want to see how it really is controlled.
				/*if dance > 0 {
					dance = dance - 1
				}*/
				timeSinceLastDecrement := time.Since(timeOfLastDecrement)
				if timeSinceLastDecrement > decrementDuration {
					dance = dance - 1
					timeOfLastDecrement = time.Now()
				}

				continue
			}

			status = "Motion detected"
			statusColor = color.RGBA{255, 0, 0, 0}
			gocv.DrawContours(&img, contours, i, statusColor, 2)

			rect := gocv.BoundingRect(contours.At(i))
			gocv.Rectangle(&img, rect, color.RGBA{0, 0, 255, 0}, 2)

			timeSinceLastIncrement := time.Since(timeOfLastIncrement)
			if timeSinceLastIncrement > incrementDuration {
				dance = dance + 1
				timeOfLastIncrement = time.Now()
			}
		}

		gocv.PutText(&img, status, image.Pt(10, 20), gocv.FontHersheyPlain, 1.2, statusColor, 2)
		gocv.PutText(&img, "Dance = "+strconv.Itoa(dance), image.Pt(10, 40), gocv.FontHersheyPlain, 1.2, statusColor, 2)

		window.IMShow(img)
		if window.WaitKey(1) == 27 {
			break
		}
	}
}
