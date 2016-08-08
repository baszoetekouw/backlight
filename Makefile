
GOCC=gccgo-6
GOCCFLAGS=-O2
LDFLAGS=

all: backlight

clean:
	rm -f *.o
	rm -f backlight

backlight: backlight.o

%: %.o
	$(GOCC) $(LDFLAGS) $< -o $@

%.o: %.go
	$(GOCC) $(GOCCFLAGS) -c $< -o $@

.PHONY: clean all
