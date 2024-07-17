# Gospat - Spatial Data Analysis with Go

This repository contains experimental projects using the  [Go programming language](https://go.dev) for spatial data analysis. This is evidently a work in progress, with future developments anticipated (but then reality comes).

With Go language installed, just run make to build executable files for your platform.

## Current Features

### s2rgb - Sentinel-2 to True Color RGB Converter

A tool to convert Sentinel-2 SAFE images to true color RGB (TIF format). This tool uses  [godal](https://github.com/airbusgeo/godal) for raster processing. It maps bands 4, 3, and 2 from the Sentinel image to Red, Green, and Blue, respectively, maintaining the original 10m resolution.

This tool implements three methods for defining the range for RGB conversion of each band:

1. percentiles (default method): Calculates the percentiles of each band and uses the 2% and 98% values as limits for the band conversion to 8-bit. This method can be adjusted using the `-lower` and `-upper` arguments to define different percentiles within the range 0-1.

2. sdevs: Uses the mean ± standard deviation of the band to determine the limits for conversion. By default, it uses 1.96 standard deviations, but this can be adjusted using the `-sdevs` argument.

3. minmax: Detects the minimum and maximum values of each band and linearly converts them to 8-bit.

Usage of the tools is straightforward:

`bin/s2rgb [OPTIONS] SENTINEL-2-SAFE-FILE OUTPUT.tif`

Check `bin/s2rgb -h` for more information about the available arguments.



