/*
 * Copyright 2019 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

 package main

 import (
	 "context"
	 "flag"
	 "fmt"
	 "log"
	 "net/http"
	 "os"
	 "time"
	 "net/url"
	 b64 "encoding/base64"
	 "encoding/json"
 
	 eventingv1alpha1 "github.com/knative/eventing/pkg/apis/eventing/v1alpha1"
	 "github.com/knative/eventing/pkg/provisioners"
	 "github.com/knative/pkg/signals"
	 "go.uber.org/zap"
	 "sigs.k8s.io/controller-runtime/pkg/client/config"
	 "sigs.k8s.io/controller-runtime/pkg/manager"
	 //"github.com/knative/eventing/pkg/reconciler/names"
 )
 
 var (
	 port = 8080
 
	 readTimeout  = 1 * time.Minute
	 writeTimeout = 1 * time.Minute
 )
 
 func main() {
	 logConfig := provisioners.NewLoggingConfig()
	 logger := provisioners.NewProvisionerLoggerFromConfig(logConfig).Desugar()
	 defer logger.Sync()
	 flag.Parse()
 
	 logger.Info("Starting...")
	 
 
	 mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	 if err != nil {
		 logger.Fatal("Error starting up.", zap.Error(err))
	 }
 
	 if err = eventingv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		 logger.Fatal("Unable to add eventingv1alpha1 scheme", zap.Error(err))
	 }
 
	 //f := getRequiredEnv("FRESHBOOKS_CHANNEL_URL")

	 c := getRequiredEnv("GITHUB_CHANNEL_URL")


	 //c := domainToURL(names.ServiceHostName("sink-channel", "default"))

	 // c := "http://freshbooks-channel-channel-m76w9.default.svc.cluster.local/"

	 
	 h := NewHandler(logger, c)
 
	 s := &http.Server{
		 Addr:         fmt.Sprintf(":%d", port),
		 Handler:      h,
		 ErrorLog:     zap.NewStdLog(logger),
		 ReadTimeout:  readTimeout,
		 WriteTimeout: writeTimeout,
	 }
 
	 err = mgr.Add(&runnableServer{
		 logger: logger,
		 s:      s,
	 })
	 if err != nil {
		 logger.Fatal("Unable to add runnableServer", zap.Error(err))
	 }
 
	 // Set up signals so we handle the first shutdown signal gracefully.
	 stopCh := signals.SetupSignalHandler()
	 // Start blocks forever.
	 if err = mgr.Start(stopCh); err != nil {
		 logger.Error("manager.Start() returned an error", zap.Error(err))
	 }
	 logger.Info("Exiting...")
 
	 ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	 defer cancel()
	 if err = s.Shutdown(ctx); err != nil {
		 logger.Error("Shutdown returned an error", zap.Error(err))
	 }
 }

 func domainToURL(domain string) string {
	u := url.URL{
		Scheme: "http",
		Host:   domain,
		Path:   "/",
	}
	return u.String()
}
 
 func getRequiredEnv(envKey string) string {
	 val, defined := os.LookupEnv(envKey)
	 if !defined {
		 log.Fatalf("required environment variable not defined '%s'", envKey)
	 }
	 return val
 }
 
 // http.Handler that takes a single request in and sends it out to a single destination.
 type Handler struct {
	 receiver    *provisioners.MessageReceiver
	 dispatcher  *provisioners.MessageDispatcher
	 destination string
 
	 logger *zap.Logger
 }
 
 // NewHandler creates a new ingress.Handler.
 func NewHandler(logger *zap.Logger, destination string) *Handler {
	 handler := &Handler{
		 logger:      logger,
		 dispatcher:  provisioners.NewMessageDispatcher(logger.Sugar()),
		 destination: destination,
	 }
	 // The receiver function needs to point back at the handler itself, so set it up after
	 // initialization.
	 handler.receiver = provisioners.NewMessageReceiver(createReceiverFunction(handler), logger.Sugar())
 
	 return handler
 }
 
 func createReceiverFunction(f *Handler) func(provisioners.ChannelReference, *provisioners.Message) error {
	 return func(_ provisioners.ChannelReference, m *provisioners.Message) error {
		 // TODO Filter.
		 return f.dispatch(m)
	 }
 }
 
 // http.Handler interface.
 func (f *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	 f.receiver.HandleRequest(w, r)
 }

 type pingDataFormat struct {
    ID          string
    Data int
    uploadEndTimeInSeconds   int
    callbackURL              string 
}
 
 // dispatch takes the request, and sends it out the f.destination. If the dispatched
 // request returns successfully, then return nil. Else, return an error.
 func (f *Handler) dispatch(msg *provisioners.Message) error {
	f.logger.Info("Message to be sent = ", zap.Any("msg", msg))
	 f.logger.Info("Message Payload", zap.Any("payload", string(msg.Payload[:])))
	
	 jStr := string(msg.Payload[:])
	 fmt.Println(jStr)
 
	 type Payload struct {
		 Id string `json:"id"`
		 Data string `json:"data"`
	 }
	 
	 var payload Payload
 
	 json.Unmarshal([]byte(jStr), &payload)
	 fmt.Printf("%+v\n", payload.Data)
	 dataDec, _  := b64.StdEncoding.DecodeString(payload.Data)
	 fmt.Printf("%+v\n", string(dataDec))
	 
	 type Data struct {
		 Source string `json:"source"`
		 Type string `json:"type"`
	 }
	 
	 var data Data
	 
	 
	 json.Unmarshal([]byte(string(dataDec)), &data)
	 fmt.Printf("%+v\n", data)
	 var destination = f.destination
	 if data.Source == "GITHUB" {
		destination = getRequiredEnv("GITHUB_CHANNEL_URL")
	 } else if data.Source == "FRESHBOOKS" {
		 destination = getRequiredEnv("FRESHBOOKS_CHANNEL_URL")
	 } else {
		 destination = getRequiredEnv("COMMON_CHANNEL_URL")
	 }
	 fmt.Printf("%+v\n", destination)

	 err := f.dispatcher.DispatchMessage(msg, destination, "", provisioners.DispatchDefaults{})
	 if err != nil {
		 f.logger.Error("Error dispatching message", zap.String("destination", destination))
		 f.logger.Error("Error received", zap.Error(err))
	 }
	 return err
 }
 
 // runnableServer is a small wrapper around http.Server so that it matches the manager.Runnable
 // interface.
 type runnableServer struct {
	 logger *zap.Logger
	 s      *http.Server
 }
 
 func (r *runnableServer) Start(<-chan struct{}) error {
	 r.logger.Info("Ingress Listening...", zap.String("Address", r.s.Addr))
	 return r.s.ListenAndServe()
 }