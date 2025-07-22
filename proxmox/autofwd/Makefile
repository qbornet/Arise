RM := rm
NAME := autofwd-daemon


$(NAME): main.go
	go build -o $@ .


build: $(NAME)

clean:
	$(RM) $(NAME)

re: clean build

.PHONY: build clean
