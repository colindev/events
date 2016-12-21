KEEP ?= n
CONTENT ?=

conn:
	if [ "$(KEEP)" = "n" ] ; then\
		cat $(CONTENT) | nc 127.0.0.1 8000;\
	else\
		cat $(CONTENT) - | nc 127.0.0.1 8000;\
	fi

