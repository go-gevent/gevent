FROM golang

WORKDIR /app

RUN apt update && apt install -y git autoconf automake libtool bison flex gcc g++ vim telnet

RUN git clone https://github.com/satori-com/tcpkali.git && \
    cd tcpkali && \
    test -f configure || autoreconf -iv && \
    ./configure && \
    make && \
    make install

COPY . .

RUN go mod tidy
