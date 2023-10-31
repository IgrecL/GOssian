# GOssian

TCP client/server session where clients can send an image and receive a blurred version thanks to a fast multi-threaded implementation of Gaussian blur in Golang.

<p align="center">
  <img src="https://github.com/IgrecL/GOssian/assets/99618877/8a05c799-7e5d-4bdc-b317-432f614514d0" alt="Gaussian" width="80%">
</p>

## How to use

### Server side

To run the server you just have to run Server or Server.exe depending on your OS.<br>
The server stores the images sent by its clients with the date of the request and the client IP.

### Client side

Steps:
1. Run the binary (Client or Client.exe depending on your OS).
2. A prompt asks you which image you want to send. You can either write the relative or absolute path to this file, or type "exit" to close the server.
3. Then you are asked to give the following processing parameters:
    * `radius` [int] – The radius of the mask (kernel) used for the Gaussian blur. The higher it is the more 'precise' the blur is, but big values lead to speed reduction. We recommend using values between 3 and 10.
    * `intensity` [float] – The intensity of the blur, which corresponds to the standard deviation given to the Gaussian function. The bigger your image is the bigger this value should be.
    * `compression` [int] – The JPEG compression percentage.
4. The processed image is created in the current directory with the date as its name.

For instance, good parameters for a 1920x1080 image are: radius = 6, intensity = 5.0, compression = 100.

## How it works

This implementation uses the real Gaussian blur algorithm, unlike image processing software like Photoshop which use approximations for better performances.

The Gaussian blur uses a 2D normal distribution for calculating the transformation to apply to each pixel in the image.
Values from this distribution are used to build a convolution matrix which is applied to the original image.
Each pixel's new value is set to a weighted average of that pixel's neighborhood, which results in a Gaussian blur.

<img src="https://user-images.githubusercontent.com/99618877/216345476-a3a321a3-544d-4843-9f93-afeb4bdf09a2.png" width="200">

Pixels are processed by different goroutines from a goroutine pool, which allows to drastically reduce the execution time.
Tests on a 8 core computer with a 1920x1080 image with the above recommended parameters gave the following results:
* without goroutines: 12.875s
* with goroutines: 3.872s
