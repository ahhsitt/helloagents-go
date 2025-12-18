// Package otel provides observability integration for HelloAgents
package otel

import (
	"context"
	"time"

	"github.com/easyops/helloagents-go/pkg/core/llm"
	"github.com/easyops/helloagents-go/pkg/core/message"
	"go.opentelemetry.io/otel/attribute"
)

// TracedProvider wraps an LLM provider with tracing support
type TracedProvider struct {
	provider llm.Provider
	tracer   Tracer
	metrics  Metrics
}

// TracedProviderOption configures the traced provider
type TracedProviderOption func(*TracedProvider)

// WithTracedProviderTracer sets the tracer
func WithTracedProviderTracer(tracer Tracer) TracedProviderOption {
	return func(p *TracedProvider) {
		p.tracer = tracer
	}
}

// WithTracedProviderMetrics sets the metrics
func WithTracedProviderMetrics(metrics Metrics) TracedProviderOption {
	return func(p *TracedProvider) {
		p.metrics = metrics
	}
}

// NewTracedProvider creates a traced LLM provider wrapper
func NewTracedProvider(provider llm.Provider, opts ...TracedProviderOption) *TracedProvider {
	tp := &TracedProvider{
		provider: provider,
		tracer:   NewNoopTracer(),
		metrics:  NewNoopMetrics(),
	}

	for _, opt := range opts {
		opt(tp)
	}

	return tp
}

// Generate generates a response with tracing
func (p *TracedProvider) Generate(ctx context.Context, req llm.Request) (llm.Response, error) {
	ctx, span := p.tracer.Start(ctx, "llm.generate",
		WithSpanKind(SpanKindClient),
		WithAttributes(
			LLMProvider(p.provider.Name()),
			LLMModel(p.provider.Model()),
		),
	)
	defer span.End()

	startTime := time.Now()

	// Execute the request
	resp, err := p.provider.Generate(ctx, req)
	duration := time.Since(startTime)

	// Record metrics
	p.recordMetrics(ctx, resp, err, duration)

	// Update span
	if err != nil {
		span.RecordError(err)
		span.SetStatus(StatusError, err.Error())
		return resp, err
	}

	// Add token usage to span
	span.SetAttributes(
		attribute.Int(AttrLLMPromptTokens, resp.TokenUsage.PromptTokens),
		attribute.Int(AttrLLMCompletionTokens, resp.TokenUsage.CompletionTokens),
		attribute.Int(AttrLLMTotalTokens, resp.TokenUsage.TotalTokens),
	)
	span.AddEvent("llm.response",
		attribute.String("finish_reason", resp.FinishReason),
	)
	span.SetStatus(StatusOK, "")

	return resp, nil
}

// GenerateStream generates a streaming response with tracing
func (p *TracedProvider) GenerateStream(ctx context.Context, req llm.Request) (<-chan llm.StreamChunk, <-chan error) {
	ctx, span := p.tracer.Start(ctx, "llm.generate_stream",
		WithSpanKind(SpanKindClient),
		WithAttributes(
			LLMProvider(p.provider.Name()),
			LLMModel(p.provider.Model()),
		),
	)

	chunkCh, errCh := p.provider.GenerateStream(ctx, req)

	// Wrap the channels to track completion
	tracedChunkCh := make(chan llm.StreamChunk)
	tracedErrCh := make(chan error, 1)

	go func() {
		defer close(tracedChunkCh)
		defer close(tracedErrCh)
		defer span.End()

		startTime := time.Now()
		var lastChunk llm.StreamChunk

		for {
			select {
			case chunk, ok := <-chunkCh:
				if !ok {
					// Stream ended
					duration := time.Since(startTime)
					if lastChunk.TokenUsage != nil {
						span.SetAttributes(
							attribute.Int(AttrLLMPromptTokens, lastChunk.TokenUsage.PromptTokens),
							attribute.Int(AttrLLMCompletionTokens, lastChunk.TokenUsage.CompletionTokens),
							attribute.Int(AttrLLMTotalTokens, lastChunk.TokenUsage.TotalTokens),
						)
						// Record metrics
						p.metrics.Counter(MetricLLMRequests).Add(ctx, 1,
							NewAttr("provider", p.provider.Name()),
							NewAttr("model", p.provider.Model()),
							NewAttr("status", "success"),
						)
						p.metrics.Counter(MetricLLMTokensPrompt).Add(ctx, int64(lastChunk.TokenUsage.PromptTokens),
							NewAttr("provider", p.provider.Name()),
							NewAttr("model", p.provider.Model()),
						)
						p.metrics.Counter(MetricLLMTokensCompletion).Add(ctx, int64(lastChunk.TokenUsage.CompletionTokens),
							NewAttr("provider", p.provider.Name()),
							NewAttr("model", p.provider.Model()),
						)
					}
					p.metrics.Histogram(MetricLLMRequestDuration).Record(ctx, duration.Seconds()*1000,
						NewAttr("provider", p.provider.Name()),
						NewAttr("model", p.provider.Model()),
					)
					span.SetStatus(StatusOK, "")
					return
				}
				lastChunk = chunk
				tracedChunkCh <- chunk

			case err, ok := <-errCh:
				if ok && err != nil {
					span.RecordError(err)
					span.SetStatus(StatusError, err.Error())
					p.metrics.Counter(MetricLLMRequests).Add(ctx, 1,
						NewAttr("provider", p.provider.Name()),
						NewAttr("model", p.provider.Model()),
						NewAttr("status", "error"),
					)
					p.metrics.Counter(MetricLLMErrors).Add(ctx, 1,
						NewAttr("provider", p.provider.Name()),
						NewAttr("model", p.provider.Model()),
					)
					tracedErrCh <- err
					return
				}
			}
		}
	}()

	return tracedChunkCh, tracedErrCh
}

// Embed generates embeddings with tracing
func (p *TracedProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	ctx, span := p.tracer.Start(ctx, "llm.embed",
		WithSpanKind(SpanKindClient),
		WithAttributes(
			LLMProvider(p.provider.Name()),
			LLMModel(p.provider.Model()),
			attribute.Int("input_count", len(texts)),
		),
	)
	defer span.End()

	startTime := time.Now()
	result, err := p.provider.Embed(ctx, texts)
	duration := time.Since(startTime)

	// Record metrics
	p.metrics.Histogram(MetricLLMRequestDuration).Record(ctx, duration.Seconds()*1000,
		NewAttr("provider", p.provider.Name()),
		NewAttr("operation", "embed"),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(StatusError, err.Error())
		p.metrics.Counter(MetricLLMErrors).Add(ctx, 1,
			NewAttr("provider", p.provider.Name()),
			NewAttr("operation", "embed"),
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("output_count", len(result)))
	span.SetStatus(StatusOK, "")
	return result, nil
}

// Name returns the provider name
func (p *TracedProvider) Name() string {
	return p.provider.Name()
}

// Model returns the model name
func (p *TracedProvider) Model() string {
	return p.provider.Model()
}

// Close closes the underlying provider
func (p *TracedProvider) Close() error {
	return p.provider.Close()
}

// recordMetrics records LLM call metrics
func (p *TracedProvider) recordMetrics(ctx context.Context, resp llm.Response, err error, duration time.Duration) {
	if err != nil {
		p.metrics.Counter(MetricLLMRequests).Add(ctx, 1,
			NewAttr("provider", p.provider.Name()),
			NewAttr("model", p.provider.Model()),
			NewAttr("status", "error"),
		)
		p.metrics.Counter(MetricLLMErrors).Add(ctx, 1,
			NewAttr("provider", p.provider.Name()),
			NewAttr("model", p.provider.Model()),
		)
	} else {
		p.metrics.Counter(MetricLLMRequests).Add(ctx, 1,
			NewAttr("provider", p.provider.Name()),
			NewAttr("model", p.provider.Model()),
			NewAttr("status", "success"),
		)
		p.metrics.Counter(MetricLLMTokensPrompt).Add(ctx, int64(resp.TokenUsage.PromptTokens),
			NewAttr("provider", p.provider.Name()),
			NewAttr("model", p.provider.Model()),
		)
		p.metrics.Counter(MetricLLMTokensCompletion).Add(ctx, int64(resp.TokenUsage.CompletionTokens),
			NewAttr("provider", p.provider.Name()),
			NewAttr("model", p.provider.Model()),
		)
	}

	p.metrics.Histogram(MetricLLMRequestDuration).Record(ctx, duration.Seconds()*1000,
		NewAttr("provider", p.provider.Name()),
		NewAttr("model", p.provider.Model()),
	)
}

// TracedExecutor wraps a tool executor with tracing support
type TracedExecutor struct {
	executor ToolExecutor
	tracer   Tracer
	metrics  Metrics
}

// ToolExecutor interface for tool execution
type ToolExecutor interface {
	Execute(ctx context.Context, name string, args map[string]interface{}) ToolResult
}

// ToolResult represents a tool execution result
type ToolResult interface {
	Name() string
	IsSuccess() bool
	Output() string
	Error() error
}

// NewTracedExecutor creates a traced tool executor wrapper
func NewTracedExecutor(executor ToolExecutor, tracer Tracer, metrics Metrics) *TracedExecutor {
	if tracer == nil {
		tracer = NewNoopTracer()
	}
	if metrics == nil {
		metrics = NewNoopMetrics()
	}
	return &TracedExecutor{
		executor: executor,
		tracer:   tracer,
		metrics:  metrics,
	}
}

// Execute executes a tool with tracing
func (e *TracedExecutor) Execute(ctx context.Context, name string, args map[string]interface{}) ToolResult {
	ctx, span := e.tracer.Start(ctx, "tool.execute",
		WithSpanKind(SpanKindInternal),
		WithAttributes(
			ToolName(name),
		),
	)
	defer span.End()

	startTime := time.Now()
	result := e.executor.Execute(ctx, name, args)
	duration := time.Since(startTime)

	// Record metrics
	status := "success"
	if !result.IsSuccess() {
		status = "error"
		span.RecordError(result.Error())
		span.SetStatus(StatusError, result.Error().Error())
		e.metrics.Counter(MetricToolErrors).Add(ctx, 1,
			NewAttr("tool", name),
		)
	} else {
		span.SetStatus(StatusOK, "")
	}

	e.metrics.Counter(MetricToolCalls).Add(ctx, 1,
		NewAttr("tool", name),
		NewAttr("status", status),
	)
	e.metrics.Histogram(MetricToolCallDuration).Record(ctx, duration.Seconds()*1000, // milliseconds
		NewAttr("tool", name),
	)
	span.SetAttributes(ToolDuration(duration.Milliseconds()))

	return result
}

// AgentTracer provides helper functions for tracing agent operations
type AgentTracer struct {
	tracer  Tracer
	metrics Metrics
}

// NewAgentTracer creates a new agent tracer
func NewAgentTracer(tracer Tracer, metrics Metrics) *AgentTracer {
	if tracer == nil {
		tracer = NewNoopTracer()
	}
	if metrics == nil {
		metrics = NewNoopMetrics()
	}
	return &AgentTracer{
		tracer:  tracer,
		metrics: metrics,
	}
}

// StartRun starts a trace span for an agent run
func (at *AgentTracer) StartRun(ctx context.Context, agentName, agentType string) (context.Context, Span) {
	return at.tracer.Start(ctx, "agent.run",
		WithSpanKind(SpanKindInternal),
		WithAttributes(
			AgentName(agentName),
			AgentType(agentType),
		),
	)
}

// RecordIteration records an agent iteration event
func (at *AgentTracer) RecordIteration(ctx context.Context, iteration, maxIterations int) {
	span := at.tracer.SpanFromContext(ctx)
	span.AddEvent("agent.iteration",
		attribute.Int(AttrAgentIteration, iteration),
		attribute.Int(AttrAgentMaxIter, maxIterations),
	)
}

// RecordToolCall records a tool call event
func (at *AgentTracer) RecordToolCall(ctx context.Context, toolName string, success bool, durationMs int64) {
	span := at.tracer.SpanFromContext(ctx)
	span.AddEvent("agent.tool_call",
		attribute.String(AttrToolName, toolName),
		attribute.Bool("success", success),
		attribute.Int64(AttrToolDuration, durationMs),
	)
}

// RecordTokenUsage records token usage metrics
func (at *AgentTracer) RecordTokenUsage(ctx context.Context, usage message.TokenUsage, provider, model string) {
	at.metrics.Counter(MetricLLMTokensPrompt).Add(ctx, int64(usage.PromptTokens),
		NewAttr("provider", provider),
		NewAttr("model", model),
	)
	at.metrics.Counter(MetricLLMTokensCompletion).Add(ctx, int64(usage.CompletionTokens),
		NewAttr("provider", provider),
		NewAttr("model", model),
	)
}

// FinishRun finishes the agent run span
func (at *AgentTracer) FinishRun(span Span, err error, durationMs int64) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(StatusError, err.Error())
	} else {
		span.SetStatus(StatusOK, "")
	}
	span.SetAttributes(attribute.Int64("duration_ms", durationMs))
	span.End()
}

// compile-time interface check
var _ llm.Provider = (*TracedProvider)(nil)
