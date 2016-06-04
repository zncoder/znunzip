# zniconv

Package zniconv provides a Reader to convert the charset of data.
It wraps an io.Reader, and converts the data read from the io.Reader to the target charset.
The actual conversion is done by the glibc iconv.

See main/zniconv.go for example.

To install the zniconv package,

    go get github.com/zncoder/zniconv

To install the zniconv binary,

    cd main
    go install zniconv.go

