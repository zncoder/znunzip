# znunzip

znunzip is a simple tool to unzip zip files. It uses zniconv to decode non-UTF8 file names.

To install,

    go get github.com/zncoder/znunzip

For example to unzip a zip file that contains file names with chinese characters,

    znunzip -c=gbk -x foo.zip
