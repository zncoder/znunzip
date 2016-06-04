# znunzip

znunzip is a simple tool to unzip zip files. It uses zniconv to decode non-UTF8 file names.

To install,

    go get github.com/zncoder/znunzip

To test a zip file,

    znunzip foo.zip

To unzip a zip file,

    znunzip -x foo.zip

To unzip a zip file that has file names in chinese characters,

    znunzip -c gb18030 -x foo.zip


