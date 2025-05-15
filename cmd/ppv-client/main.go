package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/yasushisakai/ppv-service/client"
)

func main() {
	// get args
	addr := flag.String("address", "localhost", "server address")
	port := flag.String("port", "50051", "server port")
	mat := flag.String("matrix", "", "comma separated floats")
	nDel := flag.Int("delegates", 2, "number of delegates")
	nPol := flag.Int("policies", 2, "number of policies")
	nInt := flag.Int("intermediaries", 0, "number of intermediates")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout")

	flag.Parse()

	parts := strings.Split(*mat, ",")
	matrix := make([]float64, len(parts))

	for i, s := range parts {
		fmt.Sscan(s, &matrix[i])
	}

	// TODO: This should checked in the server as well
	log.Printf("[DEBUG] matrix: %v", matrix)
	log.Printf("[DEBUG] node num: %d", *nDel+*nPol+*nInt)
	if (*nDel + *nPol + *nInt) != int(math.Sqrt(float64(len(matrix)))) {
		log.Fatalf("[ERROR] matrix size mismatch")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client, err := client.New(*addr, *port)

	if err != nil {
		log.Fatal(err)
	}

	consensus, influence, iteration, didConverge, err := client.Compute(ctx, matrix, *nDel, *nPol, *nInt)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Consensus:", consensus)
	fmt.Println("Influence:", influence)
	fmt.Println("Iteration:", iteration)
	fmt.Println("Did Converge:", didConverge)
}
