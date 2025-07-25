RM := rm
LOG := autofwd-daemon.log
NAME := autofwd-daemon


$(NAME): main.go
	go build -o $@ .


build: $(NAME)

clean:
	$(RM) $(NAME)

clean-log:
	$(RM) $(LOG)

re: clean clean-log build

.PHONY: build clean clean-log
