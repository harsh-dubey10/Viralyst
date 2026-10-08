import React from 'react';
import { render, screen } from '@testing-library/react';
import AnalysisPanel from './AnalysisPanel';
import { Analysis } from '../types';

const base: Analysis = {
  audience_topics: ['coffee'],
  audience_age: '',
  audience_regions: [],
  content_topics: ['coffee'],
  audience_fit: 100,
  hook_score: 60,
  hook_signals: [],
  recommendations: [],
  hook_skip_rate: 40,
  relevant_skip_rate: 30,
  in_target_engagement: 50,
  out_target_engagement: 10,
  amplification: 2,
  engine: 'llm',
  verdict_distribution: {},
  confidence: -1,
  replicates: 0,
};

test('shows what was read from the video', () => {
  render(<AnalysisPanel analysis={{ ...base, video_hook: 'Spoken: "Stop buying pods"' }} />);
  expect(screen.getByText(/Read from your video/i)).toBeInTheDocument();
  expect(screen.getByText(/Stop buying pods/)).toBeInTheDocument();
});

test('hides the video section when there is no video', () => {
  render(<AnalysisPanel analysis={base} />);
  expect(screen.queryByText(/Read from your video/i)).not.toBeInTheDocument();
});

test('says how topics were read', () => {
  render(<AnalysisPanel analysis={{ ...base, topic_source: 'llm' }} />);
  expect(screen.getByText('Topics read by LLM')).toBeInTheDocument();
});

test('says when the hook was rated by the LLM', () => {
  render(<AnalysisPanel analysis={{ ...base, hook_source: 'llm', hook_signals: ['clear payoff'] }} />);
  expect(screen.getByText(/Rated by LLM for this audience · clear payoff/)).toBeInTheDocument();
});
