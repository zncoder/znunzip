# znunzip

znunzip is a simple tool to unzip zip files. It uses zniconv to decode non-UTF8 file names.

To install,

    go get github.com/zncoder/znunzip

Usage:

    znunzip -c=gbk -x foo.zip
