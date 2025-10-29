# Streaming Metrics Interpretation Guide

This guide explains how to interpret the streaming analysis metrics from `gocurl --streaming`.

## Quick Reference: Buffered vs Progressive Delivery

| Characteristic | Progressive Streaming | Buffered Delivery |
|----------------|----------------------|-------------------|
| **Server Processing** | Low (< 200ms) | High (> 500ms) |
| **Content Transfer** | High (seconds) | Low (milliseconds) |
| **Coefficient of Variation (CV)** | > 0.5 (variable) | ≈ 0 (uniform) |
| **Mean Delay** | > 5ms | ≈ 0ms |
| **Pattern** | burst, moderate, stalled | steady |
| **Total Chunks** | Many (> 50) | Few (< 20) |

## Understanding the Metrics

### Coefficient of Variation (CV)

**What it is:** The ratio of standard deviation to mean (σ/μ) for inter-chunk delays.

**How to interpret:**
- **CV ≈ 0**: All chunks arrived with identical timing (likely buffered)
- **CV < 0.3**: Very consistent delivery (steady streaming)
- **CV 0.3-0.7**: Moderate variation (normal streaming)
- **CV > 0.7**: High variation (bursty or stalled)

**Example:**
```
CV: 7.14  →  High variation, bursty progressive delivery
CV: 0.00  →  Zero variation, likely buffered
```

### Mean Inter-Chunk Delay

**What it is:** Average time between consecutive chunks (in milliseconds).

**How to interpret:**
- **0ms**: Chunks sent back-to-back instantly (buffered)
- **1-10ms**: Fast progressive delivery
- **10-50ms**: Normal progressive delivery
- **> 50ms**: Slow delivery or network issues

**Example:**
```
Mean delay: 6.50ms  →  Progressive delivery with small gaps
Mean delay: 0.00ms  →  All chunks sent instantly (buffered)
```

### Standard Deviation

**What it is:** How much inter-chunk delays vary from the mean.

**How to interpret:**
- **0ms**: No variation (perfectly uniform, likely buffered)
- **Low (< 10ms)**: Consistent delivery
- **High (> 50ms)**: Inconsistent delivery with bursts or stalls

### Min/Max Delay Range

**What it is:** The shortest and longest delays between chunks.

**How to interpret:**
- **Range: 0ms - 0ms**: All chunks instant (buffered)
- **Range: 0ms - 500ms**: Variable delivery with occasional long gaps
- **Range: 50ms - 200ms**: Consistently paced delivery

## Delivery Patterns

### Pattern: steady
- CV < 0.3
- Chunks arrive at consistent intervals
- Indicates reliable streaming or buffered delivery with uniform pacing

### Pattern: moderate
- CV 0.3-0.7
- Some variation but generally consistent
- Normal progressive streaming

### Pattern: burst
- CV > 0.7, few long delays
- Chunks arrive in bursts with gaps between
- Common with HTTP/2, network buffering, or slow generation

### Pattern: stalled
- CV > 0.7, many long delays (> 500ms)
- Frequent pauses in delivery
- May indicate server issues or processing delays

### Pattern: insufficient_data
- Only 1 chunk received
- Cannot determine pattern
- Common for small responses

## Real-World Examples

### Example 1: Progressive Streaming (Server-Sent Events)
```
Server Processing: 123ms
Content Transfer: 916ms
Total Chunks: 140
Pattern: burst
CV: 7.14
Mean delay: 6.50ms
Std deviation: 46.47ms
Range: 0ms - 500ms

Interpretation:
- Server responds quickly (123ms)
- Content delivered progressively over 916ms
- High CV (7.14) indicates variable timing
- 140 chunks show true progressive delivery
- Occasional 500ms gaps (see Range)
```

### Example 2: Buffered Delivery (Edge Cache)
```
Server Processing: 872ms
Content Transfer: 2ms
Total Chunks: 60
Pattern: steady
CV: 0.00
Mean delay: 0.00ms
Std deviation: 0.00ms
Range: 0ms - 0ms

Interpretation:
- Server takes long time (872ms) to prepare response
- Content delivered instantly in 2ms (buffered)
- CV of 0 confirms all chunks sent together
- Zero delays confirm buffering
- Content was fully prepared before transmission
```

### Example 3: Small Response
```
Total Chunks: 1
Pattern: insufficient_data
CV: 0.00
Confidence: 0%

Interpretation:
- Entire response in single chunk
- Too small to determine streaming behavior
- Normal for responses < 16KB
```

## Buffering Detection

The `buffering_detected` field uses multiple signals:

**Buffering is detected when 2+ of these are true:**
1. Only 1 chunk delivered (entire response at once)
2. High TTFB (> 1s) + burst pattern
3. Very high first chunk gap (> 1s)
4. Low CV (< 0.3) + high TTFB (> 500ms)

**Note:** `buffering_detected: false` doesn't always mean progressive streaming. Check the timing breakdown:
- **Progressive:** Low server processing, high content transfer
- **Buffered:** High server processing, low content transfer

## How to Use This Information

### For API Monitoring
- **CV > 0.5 + high content transfer time**: Progressive streaming working correctly
- **CV ≈ 0 + high server processing**: Buffering layer detected
- **High max delay**: Investigate stalls or network issues

### For Performance Optimization
- **High CV with low throughput**: Check network or TCP window scaling
- **Zero CV with high server time**: Consider removing buffering layer
- **Many stalls**: Investigate server processing or generation speed

### For A/B Testing
Compare the CV and timing breakdown:
```bash
# Route A
CV: 7.14, Server: 123ms, Transfer: 916ms  → Progressive

# Route B
CV: 0.00, Server: 872ms, Transfer: 2ms    → Buffered

# Route A has true progressive delivery
```

## Confidence Score

**What it is:** How reliable the analysis is (0-100%).

**Factors:**
- **Sample size**: More chunks = higher confidence
- **Pattern clarity**: Clear patterns (steady, buffered) = higher confidence

**Thresholds:**
- **0-50%**: Low confidence (< 5 chunks)
- **60-70%**: Medium confidence (5-10 chunks)
- **80-100%**: High confidence (> 10 chunks, clear pattern)

## Tips

1. **Look at timing breakdown first**: Server vs Content Transfer tells you the most
2. **CV is the key metric**: Separates buffered (0) from progressive (> 0.5)
3. **Check total chunks**: Few chunks might not be buffering, just small response
4. **Use JSON output for automation**: Parse metrics programmatically
5. **Compare routes**: Use same URL with different headers/backends to A/B test

## Command Examples

```bash
# Analyze streaming behavior
gocurl --streaming https://api.example.com/stream

# Compare two routes
gocurl --streaming -H 'x-backend: A' https://api.com/data -o json | jq .streaming.buffering_analysis
gocurl --streaming -H 'x-backend: B' https://api.com/data -o json | jq .streaming.buffering_analysis

# CI/CD validation (fails if buffered)
gocurl --expect-streaming https://api.com/events || exit 1
```

## Further Reading

- [Coefficient of Variation (Wikipedia)](https://en.wikipedia.org/wiki/Coefficient_of_variation)
- HTTP/2 Server Push and Streaming
- TCP Window Scaling and Buffering
