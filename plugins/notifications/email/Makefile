ifeq ($(OS), Windows_NT)
	SHELL := pwsh.exe
	.SHELLFLAGS := -NoProfile -Command
	EXT = .exe
endif

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get

BINARY_NAME = notification-email$(EXT)

build: clean
	$(GOBUILD) $(LD_OPTS) $(BUILD_VENDOR_FLAGS) -o $(BINARY_NAME)

clean:
	@$(RM) $(BINARY_NAME) $(WIN_IGNORE_ERR)
