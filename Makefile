GO = go
BIN = ./bin

TARGET = s2rgb

all: $(TARGET)

$(TARGET) : cmd/s2rgb/main.go
	$(GO) build -o $(BIN)/$(TARGET) $^ 

clean:
	rm $(BIN)/*

