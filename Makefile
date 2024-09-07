build:
	@go build -o dockerly main.go rootfs.go cgroup.go

run: build
	@./dockerly


.PHONY: build run