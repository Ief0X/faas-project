package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"
    
    "faas-project/internal/models"
    "faas-project/internal/repository"
    "github.com/nats-io/nats.go"
)

type Worker struct {
    id         int
    repository repository.FunctionRepository
}

func NewWorker(id int, repo repository.FunctionRepository) *Worker {
    return &Worker{
        id:         id,
        repository: repo,
    }
}

func (w *Worker) Start(ctx context.Context, jobs <-chan models.Function, wg *sync.WaitGroup) {
    defer wg.Done()
    
    log.Printf("Trabajador %d iniciado y esperando trabajos", w.id)
    
    for {
        select {
        case <-ctx.Done():
            log.Printf("Trabajador %d apagado\n", w.id)
            return
            
        case function, ok := <-jobs:
            if !ok {
                log.Printf("Trabajador %d canal de trabajos cerrado, apagando", w.id)
                return
            }
            
            log.Printf("Trabajador %d iniciando procesamiento de función: %s (Imagen: %s)\n", w.id, function.Name, function.Image)
            
            result, err := w.executeFunction(function)
            if err != nil {
                log.Printf("Trabajador %d error ejecutando función %s: %v\n", w.id, function.Name, err)
                continue
            }
            
            log.Printf("Trabajador %d ejecutó correctamente la función %s\n", w.id, function.Name)
            
            function.LastExecution = time.Now()
            function.LastResult = result
            log.Printf("Trabajador %d actualizando estado de función %s\n", w.id, function.Name)
            if err := w.repository.Update(function); err != nil {
                log.Printf("Trabajador %d error actualizando estado de función %s: %v\n", w.id, function.Name, err)
            } else {
                log.Printf("Trabajador %d actualizó correctamente el estado de la función %s\n", w.id, function.Name)
            }
        }
    }
}

func (w *Worker) executeFunction(function models.Function) (string, error) {
    return w.repository.ExecuteFunction(function, "")
}

func main() {
    const numWorkers = 3
    
    os.Setenv("IS_WORKER", "true")
    
    log.Printf("Iniciando servicio de trabajador con %d trabajadores", numWorkers)
    
    var repo repository.FunctionRepository
    for i := 0; i < 30; i++ {
        log.Printf("Intento %d para conectar con NATS", i+1)
        repo = repository.GetFunctionRepository()
        if repo != nil {
            log.Printf("Conectado correctamente a NATS en el intento %d", i+1)
            break
        }
        log.Printf("No se pudo conectar con NATS, esperando 2 segundos antes de volver a intentarlo...")
        time.Sleep(2 * time.Second)
    }
    
    if repo == nil {
        log.Fatal("No se pudo conectar con NATS después de varios intentos")
    }
    
    log.Printf("Repositorio inicializado correctamente")
    
    jobs := make(chan models.Function, 100)
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    var wg sync.WaitGroup
    log.Printf("Iniciando %d trabajadores...", numWorkers)
    for i := 1; i <= numWorkers; i++ {
        worker := NewWorker(i, repo)
        wg.Add(1)
        go worker.Start(ctx, jobs, &wg)
    }
    log.Printf("Todos los trabajadores iniciados correctamente")
    
    if natsRepo, ok := repo.(*repository.NATSFunctionRepository); ok {
        js := natsRepo.GetJS()
        sub, err := js.QueueSubscribe("execution.*", "workers", func(msg *nats.Msg) {
            var req struct {
                Function models.Function `json:"function"`
                Param    string         `json:"param"`
            }
            if err := json.Unmarshal(msg.Data, &req); err != nil {
                log.Printf("Error al deserializar la solicitud de ejecución: %v", err)
                return
            }
            
            result, err := repo.ExecuteFunction(req.Function, req.Param)
            if err != nil {
                result = fmt.Sprintf("error: %v", err)
            }
            
            js.Publish(fmt.Sprintf("%s.response", msg.Subject), []byte(result))
        })
        if err != nil {
            log.Fatalf("Error al suscribirse a las solicitudes de ejecución: %v", err)
        }
        defer sub.Unsubscribe()
    }
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    log.Printf("Iniciando despachador de trabajos")
    go func() {
        for {
            log.Printf("Revisando ejecuciones pendientes...")
            functions, err := repo.GetPendingExecutions()
            if err != nil {
                log.Printf("Error al obtener ejecuciones pendientes: %v\n", err)
                time.Sleep(5 * time.Second)
                continue
            }
            
            if len(functions) > 0 {
                log.Printf("Encontrados %d funciones pendientes para ejecutar", len(functions))
            }
            
            for _, function := range functions {
                log.Printf("Despachando función %s a los trabajadores", function.Name)
                jobs <- function
            }
            
            time.Sleep(1 * time.Second)
        }
    }()
    
    <-sigChan
    log.Println("Se recibió la señal de apagado")
    log.Println("Apagando trabajadores...")
    cancel()
    wg.Wait()
    log.Println("Todos los trabajadores apagados correctamente")
}