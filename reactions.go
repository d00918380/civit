package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/d00918380/civit/internal/trpc"
)

type ReactionsProcessor struct {
	trpc                               *trpc.Client
	imagesFile, modelsFile, whalesFile string
}

func (rp *ReactionsProcessor) Run() error {
	log.Println("Run started")
	for _, fn := range []func() error{
		rp.processImages,
		func() error {
			return rp.processModels(rp.modelsFile)
		},
		func() error {
			return rp.processModels(rp.whalesFile)
		},
		rp.processCompensation,
	} {
		if err := fn(); err != nil {
			return err
		}
	}
	log.Println("Run completed")
	return nil
}

func (rp *ReactionsProcessor) processImages() error {
	ctx := context.Background()
	log.Println("Processing images from", rp.imagesFile)
	f, err := os.Open(rp.imagesFile)
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := os.OpenFile("images.csv", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	ts := time.Now().Format(time.DateTime)

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		id, err := strconv.Atoi(sc.Text())
		if err != nil {
			log.Printf("Error parsing %q: %v", sc.Text(), err)
			continue
		}
		item, err := rp.trpc.Image(ctx, id)
		if err != nil {
			log.Printf("Error fetching image %d: %v", id, err)
			continue
		}
		score := score(item)
		log.Printf("Fetched image %d: %v", id, score)
		fmt.Fprintf(out, "%s,%d,%d\n", ts, id, score)
	}
	return sc.Err()
}

func (rp *ReactionsProcessor) processModels(input string) error {
	ctx := context.Background()
	log.Println("Processing models from", input)
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	defer f.Close()

	var output string
	ext := filepath.Ext(input)
	// strip extension and replace with .csv
	if ext != "" {
		output = input[:len(input)-len(ext)] + ".csv"
	}

	out, err := os.OpenFile(output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	ts := time.Now().Format(time.DateTime)

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var id int
		_, err := fmt.Sscanf(sc.Text(), "%d", &id)
		if err != nil {
			log.Printf("Error parsing %q: %v", sc.Text(), err)
			continue
		}
		model, err := rp.trpc.Model(ctx, id)
		if err != nil {
			log.Printf("Error fetching model %d: %v", id, err)
			continue
		}
		log.Printf("Fetched model %d: %v", id, model.Name)
		for _, version := range model.ModelVersions {
			log.Printf("%s %s: %d", model.Name, version.Name, version.Rank.GenerationCountAllTime)
			fmt.Fprintf(out, "%s,%q,%d\n", ts, model.Name+" "+version.Name, version.Rank.GenerationCountAllTime)
		}
	}
	return sc.Err()
}

func (rp *ReactionsProcessor) processCompensation() error {
	out, err := os.OpenFile("compensation.csv", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	ts := time.Now().Format(time.DateTime)

	comp, err := rp.trpc.CreatorProgramGetCompensationPool(context.Background())
	if err != nil {
		fmt.Println("Error fetching compensation pool:", err)
		return nil
	}
	fmt.Fprintf(out, "%s,%.2f,%.2f,%.2f\n", ts, comp.Value, comp.Size.Current, comp.Size.Forecasted)
	return nil
}

func score(i *trpc.Item) int {
	return Sum(i.Stats.LikeCountAllTime, i.Stats.LaughCountAllTime, i.Stats.HeartCountAllTime, i.Stats.CryCountAllTime)
}
