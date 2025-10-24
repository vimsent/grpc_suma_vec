package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "math/rand"
    "os"
    "sync"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "./pb"
)

type NodeConnection struct {
    conn   *grpc.ClientConn
    client pb.VectorServiceClient
    nodeID int
}

type NodeStats struct {
    reputation    float32
    correctSums   int32
    incorrectSums int32
    crashes       int32
}

func main() {
    var (
        n        = flag.Int("n", 10, "Número de sumas a realizar")
        rInit    = flag.Float64("rinit", 1000.0, "Reputación inicial")
        pFail1   = flag.Float64("pfail1", 0.1, "Probabilidad de fallo nodo 1")
        pFail2   = flag.Float64("pfail2", 0.15, "Probabilidad de fallo nodo 2")
        pFail3   = flag.Float64("pfail3", 0.2, "Probabilidad de fallo nodo 3")
        pCrash1  = flag.Float64("pcrash1", 0.05, "Probabilidad de caída nodo 1")
        pCrash2  = flag.Float64("pcrash2", 0.08, "Probabilidad de caída nodo 2")
        pCrash3  = flag.Float64("pcrash3", 0.1, "Probabilidad de caída nodo 3")
    )
    flag.Parse()
    
    rand.Seed(time.Now().UnixNano())
    
    // configurando las conexiones a los nodos
    nodes := []NodeConnection{}
    ports := []int{50051, 50052, 50053}
    
    log.Println("Conectando a los nodos...")
    for i, port := range ports {
        conn, err := grpc.Dial(
            fmt.Sprintf("localhost:%d", port),
            grpc.WithTransportCredentials(insecure.NewCredentials()),
        )
        if err != nil {
            log.Fatalf("Error conectando al nodo %d: %v", i+1, err)
        }
        defer conn.Close()
        
        nodes = append(nodes, NodeConnection{
            conn:   conn,
            client: pb.NewVectorServiceClient(conn),
            nodeID: i + 1,
        })
        log.Printf("Conectado al nodo %d en puerto %d", i+1, port)
    }
    
    log.Printf("\nIniciando %d sumas de vectores con 3 nodos", *n)
    log.Printf("Configuración - Rinicial: %.2f", *rInit)
    log.Printf("Pfallo: [%.2f, %.2f, %.2f]", *pFail1, *pFail2, *pFail3)
    log.Printf("Pcaida: [%.2f, %.2f, %.2f]\n", *pCrash1, *pCrash2, *pCrash3)
    
    // hacemso N sumas de vectores
    for i := 0; i < *n; i++ {
        vectorSize := rand.Intn(5) + 1 // Tamaño entre 1 y 5
        numVectors := rand.Intn(3) + 2 // Entre 2 y 4 vectores
        
        
        vectors := generateRandomVectors(numVectors, vectorSize)
        
        
        correctSum := calculateCorrectSum(vectors)
        
        log.Printf("\n=== Suma %d: %d vectores de tamaño %d ===", i+1, numVectors, vectorSize)
        
        
        var wg sync.WaitGroup
        for _, node := range nodes {
            wg.Add(1)
            go func(n NodeConnection) {
                defer wg.Done()
                
                ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
                defer cancel()
                
                resp, err := n.client.SumVectors(ctx, &pb.VectorRequest{Vectors: vectors})
                if err != nil {
                    log.Printf("Nodo %d: ERROR/CAÍDA - %v", n.nodeID, err)
                    return
                }
                
                
                isCorrect := compareVectors(correctSum, resp.Result.Values)
                status := "INCORRECTA"
                if isCorrect {
                    status = "CORRECTA"
                }
                log.Printf("Nodo %d: Respuesta %s", n.nodeID, status)
            }(node)
        }
        wg.Wait()
        
        
        time.Sleep(100 * time.Millisecond)
    }
    
    
    log.Printf("\n=== Obteniendo estadísticas finales ===\n")
    stats := make(map[int]*NodeStats)
    
    for _, node := range nodes {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        
        resp, err := node.client.GetStats(ctx, &pb.StatsRequest{})
        if err != nil {
            log.Printf("Error obteniendo stats del nodo %d: %v", node.nodeID, err)
            continue
        }
        
        stats[node.nodeID] = &NodeStats{
            reputation:    resp.Reputation,
            correctSums:   resp.CorrectSums,
            incorrectSums: resp.IncorrectSums,
            crashes:       resp.Crashes,
        }
        
        log.Printf("Nodo %d - Rep: %.2f, Correctas: %d, Incorrectas: %d, Caídas: %d",
            node.nodeID, resp.Reputation, resp.CorrectSums, resp.IncorrectSums, resp.Crashes)
    }
    
    // ewscribe los resultados a archivo
    writeOutput(stats)
}

func generateRandomVectors(numVectors, size int) []*pb.Vector {
    vectors := make([]*pb.Vector, numVectors)
    for i := 0; i < numVectors; i++ {
        values := make([]float32, size)
        for j := 0; j < size; j++ {
            values[j] = rand.Float32() * 100
        }
        vectors[i] = &pb.Vector{Values: values}
    }
    return vectors
}

func calculateCorrectSum(vectors []*pb.Vector) []float32 {
    if len(vectors) == 0 || len(vectors[0].Values) == 0 {
        return []float32{}
    }
    
    size := len(vectors[0].Values)
    result := make([]float32, size)
    
    for _, vector := range vectors {
        for i, val := range vector.Values {
            if i < size {
                result[i] += val
            }
        }
    }
    return result
}

func compareVectors(v1, v2 []float32) bool {
    if len(v1) != len(v2) {
        return false
    }
    
    tolerance := float32(0.001)
    for i := range v1 {
        diff := v1[i] - v2[i]
        if diff < 0 {
            diff = -diff
        }
        if diff > tolerance {
            return false
        }
    }
    return true
}

func writeOutput(stats map[int]*NodeStats) {
    file, err := os.Create("output.txt")
    if err != nil {
        log.Fatalf("Error creando archivo de salida: %v", err)
    }
    defer file.Close()
    
    fmt.Fprintf(file, "=== ESTADÍSTICAS FINALES DEL SISTEMA ===\n")
    fmt.Fprintf(file, "Fecha: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
    
    for i := 1; i <= 3; i++ {
        if s, ok := stats[i]; ok {
            fmt.Fprintf(file, "NODO %d:\n", i)
            fmt.Fprintf(file, "  Reputación Final: %.2f\n", s.reputation)
            fmt.Fprintf(file, "  Sumas Correctas: %d\n", s.correctSums)
            fmt.Fprintf(file, "  Sumas Incorrectas: %d\n", s.incorrectSums)
            fmt.Fprintf(file, "  Caídas: %d\n", s.crashes)
            fmt.Fprintf(file, "  Total de operaciones: %d\n\n", 
                s.correctSums+s.incorrectSums+s.crashes)
        }
    }
    
    log.Println("\n Resultados guardados en output.txt")
}