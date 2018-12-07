MAIN=project-milestone
SOURCES=$(MAIN).tex Makefile
PANDOC_OPTS=--filter pandoc-citeproc

all: $(MAIN).pdf

.refresh:
	touch .refresh

$(MAIN).pdf: $(MAIN).md citations.bib
	pandoc $(PANDOC_OPTS) -s $(MAIN).md -o $(MAIN).pdf

$(MAIN).rtf: $(MAIN).md citations.bib
	pandoc $(PANDOC_OPTS) -s $(MAIN).md -o $(MAIN).rtf

clean:
	rm -f $(MAIN).pdf $(MAIN).rtf
	rm -rf *~ *.tmp
	rm -f *.bbl *.blg *.aux *.end *.fls *.log *.out *.fdb_latexmk

.PHONY: clean all
