package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/d00918380/civit/internal/trpc"
)

type ReactionsProcessor struct {
	trpc       *trpc.Client
	imagesFile string
	modelsFile string
}

func (rp *ReactionsProcessor) Run() error {
	log.Println("Run started")
	if err := rp.processImages(); err != nil {
		return err
	}
	if err := rp.processModels(); err != nil {
		return err
	}
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

func (rp *ReactionsProcessor) processModels() error {
	ctx := context.Background()
	log.Println("Processing models from", rp.modelsFile)
	f, err := os.Open(rp.modelsFile)
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := os.OpenFile("models.csv", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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

func score(i *trpc.Item) int {
	return Sum(i.Stats.LikeCountAllTime, i.Stats.LaughCountAllTime, i.Stats.HeartCountAllTime, i.Stats.CryCountAllTime)
}
