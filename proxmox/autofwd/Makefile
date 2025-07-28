RM := rm
LOG := autofwd-daemon.log
NAME := autofwd-daemon


$(NAME): main.go
	CGO_ENABLED=0 go build -ldflags="-extldflags=-static" -o $@ . 
	/usr/bin/upx $@


build: $(NAME)

clean:
	$(RM) $(NAME)

clean-log:
	$(RM) $(LOG)

re: clean clean-log build

.PHONY: build clean clean-log
