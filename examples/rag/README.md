# RAG Document Q&A Example

This example demonstrates how to use the RAG (Retrieval-Augmented Generation) capabilities of HelloAgents framework to build a document-based question answering system.

## Overview

The example shows:
- Creating sample documents with metadata
- Setting up a RAG pipeline with chunking, embedding, and storage
- Ingesting documents into the vector store
- Querying the system and generating answers with source citations

## Prerequisites

- Go 1.21 or later
- OpenAI API key

## Setup

1. Set your OpenAI API key:
   ```bash
   export OPENAI_API_KEY="your-api-key"
   ```

2. Run the example:
   ```bash
   go run main.go
   ```

## Components

### Document Processing
- **Documents**: Define documents with ID, content, and metadata (source, title)
- **Chunker**: Recursive character chunker splits documents into smaller chunks (200 chars, 20 overlap)

### Storage and Retrieval
- **VectorStore**: In-memory vector store for document embeddings
- **Embedder**: LLM-based embedder using OpenAI's embedding API
- **Retriever**: Vector retriever with score threshold filtering (0.5)

### Answer Generation
- **Generator**: Custom answer generator that formats context and queries the LLM

## RAG Pipeline Flow

1. **Ingestion**:
   - Documents are loaded
   - Chunked into smaller pieces
   - Embedded into vectors
   - Stored in vector store

2. **Query**:
   - Query is embedded
   - Similar chunks are retrieved
   - Context is formatted
   - LLM generates answer with sources

## Sample Output

```
=== RAG Document Q&A Example ===

Created 5 sample documents
Ingesting documents...
Ingested documents into vector store (size: X chunks)

--- Query: What is the HelloAgents framework? ---

Answer:
HelloAgents is a Golang AI Agent framework designed for building intelligent agents...

Sources:
  1. [Score: 0.95] HelloAgents is a Golang AI Agent framework designed for building...
```

## Customization

### Custom Documents
Modify the `createSampleDocuments()` function to load your own documents.

### Different Chunking Strategy
Adjust the chunker parameters:
```go
chunker := rag.NewRecursiveCharacterChunker(
    500,  // chunk size
    50,   // overlap
)
```

### Score Threshold
Adjust the retriever threshold for stricter/looser matching:
```go
retriever := rag.NewVectorRetriever(
    store,
    embedder,
    rag.WithScoreThreshold(0.7),  // higher = stricter
)
```

## Related Examples

- [Simple Chat](../simple/README.md) - Basic agent conversation
- [ReAct Agent](../react/README.md) - Agent with tool calling
- [Memory System](../memory/README.md) - Agent with memory capabilities
