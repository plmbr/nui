// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agentindex

import "math"

// SemanticWeight scales similarity [0,1] into the lexical score space.
const SemanticWeight = 100

// MinSemanticScore is the minimum similarity required before applying a bonus.
const MinSemanticScore = 0.25

// Score returns per-doc similarity of query vs docs using in-memory TF-IDF.
// No network, no LLM — pure Go over the provided corpus.
func Score(query string, docs []Doc) map[string]float64 {
	out := make(map[string]float64, len(docs))
	if len(docs) == 0 {
		return out
	}

	docTFs := make([]map[string]float64, len(docs))
	df := map[string]float64{}
	for i, doc := range docs {
		tf := TokenizeWithFreq(doc.Text)
		docTFs[i] = tf
		seen := map[string]struct{}{}
		for term := range tf {
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			df[term]++
		}
	}

	n := float64(len(docs))
	idf := map[string]float64{}
	for term, count := range df {
		// Smooth IDF so rare terms dominate without zeros.
		idf[term] = math.Log((n+1)/(count+1)) + 1
	}

	docVecs := make([]map[string]float64, len(docs))
	docNorms := make([]float64, len(docs))
	for i, tf := range docTFs {
		vec := make(map[string]float64, len(tf))
		var norm float64
		for term, freq := range tf {
			w := (1 + math.Log(freq)) * idf[term]
			vec[term] = w
			norm += w * w
		}
		docVecs[i] = vec
		docNorms[i] = math.Sqrt(norm)
	}

	qTF := TokenizeWithFreq(query)
	qVec := make(map[string]float64, len(qTF))
	var qNorm float64
	for term, freq := range qTF {
		wIDF, ok := idf[term]
		if !ok {
			continue // only score terms present in the agent corpus
		}
		w := (1 + math.Log(freq)) * wIDF
		qVec[term] = w
		qNorm += w * w
	}
	qNorm = math.Sqrt(qNorm)
	if qNorm == 0 {
		return out
	}

	for i, doc := range docs {
		if doc.ID == "" || docNorms[i] == 0 {
			continue
		}
		var dot float64
		matched := 0
		for term, qw := range qVec {
			if dw, ok := docVecs[i][term]; ok {
				dot += qw * dw
				matched++
			}
		}
		cosine := Clamp01(dot / (qNorm * docNorms[i]))
		coverage := float64(matched) / float64(len(qVec))
		// Prefer the stronger of TF-IDF cosine and query-term coverage so
		// paraphrases that share key corpus terms still rank highly.
		out[doc.ID] = Clamp01(math.Max(cosine, coverage))
	}
	return out
}

// Cosine returns cosine similarity for dense vectors (tests / helpers).
func Cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Clamp01 maps a score into [0, 1].
func Clamp01(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// SemanticBonus converts a similarity score in [0,1] to an integer lexical bonus.
func SemanticBonus(score float64) int {
	score = Clamp01(score)
	if score < MinSemanticScore {
		return 0
	}
	return int(score*SemanticWeight + 0.5)
}
