package main

import (
	"context"
	"log"

	// Dependencies of Turbine
	"github.com/meroxa/turbine-go/pkg/turbine"
	"github.com/meroxa/turbine-go/pkg/turbine/cmd"
)

func main() {
	cmd.Start(App{})
}

var _ turbine.App = (*App)(nil)

type App struct{}

func (a App) Run(v turbine.Turbine) error {

	source, err := v.Resources("atlas-mongo")
	if err != nil {
		return err
	}

	// rr, err := source.RecordsWithContext(context.Background(), "aggregated_medicine", turbine.ConnectionOptions{
	// 	{Field: "transforms", Value: "unwrap"},
	// 	{Field: "transforms.unwrap.type", Value: "io.debezium.connector.mongodb.transforms.ExtractNewDocumentState"},
	// })
	rr, err := source.RecordsWithContext(context.Background(), "aggregated_medicine", turbine.ConnectionOptions{})
	if err != nil {
		return err
	}

	res, err := v.Process(rr, FilterStore{})
	if err != nil {
		return err
	}

	dest, err := v.Resources("west-store-mongo")
	if err != nil {
		return err
	}

	err = dest.WriteWithConfig(res, "medicine", turbine.ConnectionOptions{
		{Field: "max.batch.size", Value: "1"},
		{Field: "document.id.strategy", Value: "com.mongodb.kafka.connect.sink.processor.id.strategy.PartialValueStrategy"},
		{Field: "document.id.strategy.partial.value.projection.list", Value: "after.patient_id"},
		{Field: "document.id.strategy.partial.value.projection.type", Value: "AllowList"},
		{Field: "writemodel.strategy", Value: "com.mongodb.kafka.connect.sink.writemodel.strategy.ReplaceOneBusinessKeyStrategy"},
	})
	if err != nil {
		return err
	}

	return nil
}

type FilterStore struct{}

func (f FilterStore) Process(stream []turbine.Record) []turbine.Record {
	processedStream := make([]turbine.Record, 0)

	for _, record := range stream {
		// log.Printf("Processing record %d: %+v\n", i, record) // Logging the record details
		log.Printf("Payload: \n%s\n", record.Payload) // Logging the payload

		source := record.Payload.Get("source")
		log.Printf("source value: %v", source)
		if source == nil {
			log.Printf("Error getting source: %v", source)
			continue
		}

		if source == "atlas" {
			log.Printf("Getting storeId")
			storeId := record.Payload.Get("storeId")
			if storeId == nil {
				log.Println("error getting value: ", storeId)
				continue
			}

			if storeId == "001" {
				log.Printf("Adding record with storeId 001")
				processedStream = append(processedStream, record)
			}
		} else {
			log.Printf("Dropping record with source: %s", source)
		}
	}
	return processedStream
}
