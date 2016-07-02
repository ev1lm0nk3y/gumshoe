FROM golang:1.6.2
MAINTAINER Ryan Shatford
LABEL version="1.0"
LABEL description="Gumshoe Private Torrent Tracker"

ADD . $GOPATH/src/github.com/ev1lm0nk3y/gumshoe
RUN cd $GOPATH/src/github.com/ev1lm0nk3y/gumshoe && go install
RUN mkdir -p /gumshoe && ln -s $GOPATH/src/github.com/ev1lm0nk3y/gumshoe/www /gumshoe/www

EXPOSE 9119
CMD /go/bin/gumshoe -c $GUMSHOERC -d /gumshoe -p 9119
