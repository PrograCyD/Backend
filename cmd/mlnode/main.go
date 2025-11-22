package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"time"

	"nodosml-pc4/internal/cluster"
	"nodosml-pc4/internal/config"
	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/repository"
)

func main() {
	cfg := config.Load()
	db.InitMongo(cfg)

	addr := os.Getenv("ML_NODE_ADDR")
	if addr == "" {
		addr = ":9001"
	}

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "?"
	}

	log.Printf("[ML NODE %s] escuchando en %s", nodeID, addr)

	simsRepo := repository.NewSimilarityRepository()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		go handleConn(nodeID, conn, simsRepo)
	}
}

func handleConn(nodeID string, conn net.Conn, sims *repository.SimilarityRepository) {
	defer conn.Close()

	dec := json.NewDecoder(bufio.NewReader(conn))
	var task cluster.RecTask
	if err := dec.Decode(&task); err != nil {
		log.Printf("[ML NODE %s] decode task error: %v", nodeID, err)
		return
	}

	log.Printf("[ML NODE %s] tarea recibida: user=%d shard=%d/%d ratings=%d",
		nodeID, task.UserID, task.ShardID, task.Shards, len(task.Ratings))

	start := time.Now()

	partials, err := computeShardRecommendations(context.Background(), task, sims)
	if err != nil {
		log.Printf("[ML NODE %s] compute error: %v", nodeID, err)
		return
	}

	elapsed := time.Since(start)

	log.Printf(
		"[ML NODE %s] completado: user=%d shard=%d/%d movies_parciales=%d tiempo=%s",
		nodeID, task.UserID, task.ShardID, task.Shards, len(partials), elapsed,
	)

	resp := cluster.RecResponse{
		ShardID:  task.ShardID,
		Partials: partials,
	}

	if err := json.NewEncoder(conn).Encode(&resp); err != nil {
		log.Printf("[ML NODE %s] encode resp error: %v", nodeID, err)
	}
}

func computeShardRecommendations(
	ctx context.Context,
	task cluster.RecTask,
	simsRepo *repository.SimilarityRepository,
) ([]cluster.PartialScore, error) {

	rated := make(map[int]float64)
	for _, r := range task.Ratings {
		rated[r.MovieID] = r.Rating
	}

	scores := make(map[int]float64)
	weights := make(map[int]float64)

	for idx, r := range task.Ratings {
		if task.Shards > 0 && idx%task.Shards != task.ShardID {
			continue
		}

		neighs, err := simsRepo.GetNeighbors(ctx, r.MovieID, 100)
		if err != nil {
			return nil, err
		}

		for _, n := range neighs {
			targetID := n.MovieID
			if _, ya := rated[targetID]; ya {
				continue
			}
			if n.Sim <= 0 {
				continue
			}

			scores[targetID] += n.Sim * r.Rating
			weights[targetID] += n.Sim
		}
	}

	partials := make([]cluster.PartialScore, 0, len(scores))

	for mID, num := range scores {
		den := weights[mID]
		if den <= 0 {
			continue
		}
		partials = append(partials, cluster.PartialScore{
			MovieID: mID,
			Num:     num,
			Den:     den,
		})
	}

	return partials, nil
}
