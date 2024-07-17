GO = go
BIN = ./bin

TARGET = s2rgb

all: $(TARGET)

s2rgb : cmd/s2rgb/main.go
	$(GO) build -o $(BIN)/$@ $^ 

clean:
	rm $(BIN)/*

