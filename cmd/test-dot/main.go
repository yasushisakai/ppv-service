package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/yasushisakai/ppv-service/client"
)

func main() {
	// Create client
	c, err := client.New("localhost", "50051")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Test data: 2x2 matrices (same as in ppv_test.go and test.c)
	size := 2
	a := []float64{
		1.0, 2.0,
		3.0, 4.0,
	}
	b := []float64{
		1.0, 3.0,
		2.0, 4.0,
	}

	// Expected result (same calculation as in test.c):
	// [1 2] * [1 3] = [1*1+2*2  1*3+2*4] = [5  11]
	// [3 4]   [2 4]   [3*1+4*2  3*3+4*4]   [11 25]
	expectedDot := []float64{5.0, 11.0, 11.0, 25.0}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Testing Dot function...")
	result, err := c.Dot(ctx, a, b, size)
	if err != nil {
		log.Fatalf("Dot computation failed: %v", err)
	}

	// Verify results
	for i := 0; i < size*size; i++ {
		if math.Abs(result[i]-expectedDot[i]) > 0.001 {
			fmt.Printf("[fail] mismatch in `dot` at position %d: got %.3f, expected %.3f\n",
				i, result[i], expectedDot[i])
			os.Exit(1)
		}
	}

	fmt.Println("ok!")
}