package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "math/rand"
    "net"
    "sync"
    "time"
    
    "google.golang.org/grpc"
    pb "./pb"
)

type VectorServer struct {
    pb.UnimplementedVectorServiceServer
    nodeID       int32
    failProb     float64
    crashProb    float64
    reputation   float64
    correctSums  int32
    incorrectSums int32
    crashes      int32
    mu           sync.RWMutex
}

func NewVectorServer(nodeID int32, failProb, crashProb, initialRep float64) *VectorServer {
    return &VectorServer{
        nodeID:     nodeID,
        failProb:   failProb,
        crashProb:  crashProb,
        reputation: initialRep,
    }
}

// SumVectors implementa el RPC para sumar vectores
func (s *VectorServer) SumVectors(ctx context.Context, req *pb.VectorRequest) (*pb.VectorResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // Simular probabilidad de caída
    if rand.Float64() < s.crashProb {
        s.crashes++
        s.reputation -= 300
        log.Printf("Nodo %d: CAÍDA - Nueva reputación: %.2f", s.nodeID, s.reputation)
        return nil, fmt.Errorf("nodo %d no disponible", s.nodeID)
    }
    
    // Calcular suma correcta
    var resultSize int
    if len(req.Vectors) > 0 && len(req.Vectors[0].Values) > 0 {
        resultSize = len(req.Vectors[0].Values)
    }
    
    result := make([]float32, resultSize)
    for _, vector := range req.Vectors {
        for i, val := range vector.Values {
            if i < len(result) {
                result[i] += val
            }
        }
    }
    
    // Simular probabilidad de fallo (respuesta incorrecta)
    if rand.Float64() < s.failProb {
        // Introducir error en el resultado
        if len(result) > 0 {
            errorIndex := rand.Intn(len(result))
            result[errorIndex] += rand.Float32() * 10 - 5 // Error aleatorio
        }
        s.incorrectSums++
        penalty := float64(150 + rand.Intn(101)) // 150-250
        s.reputation -= penalty
        log.Printf("Nodo %d: Respuesta INCORRECTA - Penalización: %.2f - Nueva reputación: %.2f", 
            s.nodeID, penalty, s.reputation)
    } else {
        s.correctSums++
        reward := float64(100 + rand.Intn(81)) // 100-180
        s.reputation += reward
        log.Printf("Nodo %d: Respuesta CORRECTA - Recompensa: %.2f - Nueva reputación: %.2f", 
            s.nodeID, reward, s.reputation)
    }
    
    return &pb.VectorResponse{
        Result: &pb.Vector{Values: result},
        NodeId: s.nodeID,
    }, nil
}

// GetStats devuelve las estadísticas del nodo
func (s *VectorServer) GetStats(ctx context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    return &pb.StatsResponse{
        NodeId:        s.nodeID,
        Reputation:    float32(s.reputation),
        CorrectSums:   s.correctSums,
        IncorrectSums: s.incorrectSums,
        Crashes:       s.crashes,
    }, nil
}

func main() {
    var (
        nodeID      = flag.Int("id", 1, "ID del nodo (1, 2, o 3)")
        port        = flag.Int("port", 50051, "Puerto del servidor")
        failProb    = flag.Float64("pfail", 0.1, "Probabilidad de fallo")
        crashProb   = flag.Float64("pcrash", 0.05, "Probabilidad de caída")
        initialRep  = flag.Float64("rinit", 1000.0, "Reputación inicial")
    )
    flag.Parse()
    
    rand.Seed(time.Now().UnixNano())
    
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
    if err != nil {
        log.Fatalf("Error al escuchar: %v", err)
    }
    
    grpcServer := grpc.NewServer()
    vectorServer := NewVectorServer(int32(*nodeID), *failProb, *crashProb, *initialRep)
    pb.RegisterVectorServiceServer(grpcServer, vectorServer)
    
    log.Printf("Servidor Nodo %d iniciado en puerto %d (Pfallo=%.2f, Pcaida=%.2f, Rinicial=%.2f)",
        *nodeID, *port, *failProb, *crashProb, *initialRep)
    
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Error al servir: %v", err)
    }
} 