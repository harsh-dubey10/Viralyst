import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import App from './App';

afterEach(() => {
  jest.restoreAllMocks();
});

const runButton = () => screen.getByRole('button', { name: /Run simulation/i });

test('run is disabled until there is a niche and a hook', () => {
  render(<App />);
  expect(runButton()).toBeDisabled();
  expect(screen.getByText(/Pick a niche or describe your audience/i)).toBeInTheDocument();

  fireEvent.click(screen.getByRole('button', { name: 'Cooking' }));
  expect(screen.getByLabelText(/Your niche/i)).toHaveValue('cooking');
  expect(runButton()).toBeDisabled();

  fireEvent.change(screen.getByLabelText(/Your hook/i), { target: { value: 'This pasta takes 10 minutes.' } });
  expect(runButton()).toBeEnabled();
});

test('accepts any niche typed in', () => {
  render(<App />);
  fireEvent.change(screen.getByLabelText(/Your niche/i), { target: { value: 'aquascaping' } });
  fireEvent.change(screen.getByLabelText(/Your hook/i), { target: { value: 'My 5-gallon tank after 90 days.' } });
  expect(runButton()).toBeEnabled();
});

test('an example fills every field', () => {
  render(<App />);
  fireEvent.click(screen.getByText('Vietnam on a budget'));
  expect(screen.getByLabelText(/Your niche/i)).toHaveValue('travel');
  expect(screen.getByDisplayValue(/Budget backpackers/i)).toBeInTheDocument();
  expect(screen.getByRole('button', { name: 'Travel' })).toHaveAttribute('aria-pressed', 'true');
});

test('shows the API validation message when a simulation is rejected', async () => {
  global.fetch = jest.fn().mockResolvedValue({
    ok: false,
    status: 503,
    json: () => Promise.resolve({ detail: 'Job queue (Redis) is unavailable' }),
  }) as jest.Mock;

  render(<App />);
  fireEvent.click(screen.getByText('Weeknight pasta'));
  fireEvent.click(runButton());

  expect(await screen.findByText(/Job queue \(Redis\) is unavailable/i)).toBeInTheDocument();
  expect(runButton()).toBeEnabled();
});

test('sends the niche and shows the verdict after a completed simulation', async () => {
  const result = {
    personas: [
      { persona_id: 1, name: 'A', age: 22, location: 'Pune, India', followers: 100, interests: ['fitness'], behavior: 'sharer', in_target: true, shown: true, action: 'share', watch: 90, round: 1, why: 'x' },
      { persona_id: 2, name: 'B', age: 40, location: 'Paris, FR', followers: 80, interests: ['art'], behavior: 'lurker', in_target: false, shown: false, action: 'never', watch: 0, round: 0, why: '' },
    ],
    edges: [],
    virality: 64, algo_score: 50, engagement: 70,
    breakdown: { likes: 1, shares: 1, comments: 0, scrolls: 0 },
    verdict: { head: 'Niche hit.', sub: 'Your audience engaged.', verdict: 'Strong in-demo, no breakout', in_pct: 100, share_rate: 50 },
    analysis: {
      audience_topics: ['fitness'], audience_age: 'Millennials (28–43)', audience_regions: [],
      content_topics: ['fitness', 'nutrition'], audience_fit: 100, hook_score: 82,
      hook_signals: ['Concrete numbers / results'], recommendations: ['Add a concrete number.'],
      hook_skip_rate: 30, relevant_skip_rate: 25, in_target_engagement: 60, out_target_engagement: 10,
      amplification: 2.5, engine: 'heuristic', verdict_distribution: { 'Niche hit.': 20, 'Echo chamber.': 5 },
      confidence: 80, replicates: 25,
    },
  };
  const responses: Record<string, unknown> = {
    '/api/simulate': { job_id: 'abcdef12-0000', status: 'pending' },
    '/api/status/abcdef12-0000': { status: 'complete' },
    '/api/results/abcdef12-0000': result,
  };
  const fetchMock = jest.fn((url: string) => {
    const path = Object.keys(responses).find((k) => url.endsWith(k))!;
    return Promise.resolve({ ok: true, status: 200, json: () => Promise.resolve(responses[path]) });
  });
  global.fetch = fetchMock as unknown as typeof fetch;

  render(<App />);
  fireEvent.click(screen.getByRole('button', { name: 'Fitness' }));
  fireEvent.change(screen.getByLabelText(/Your hook/i), { target: { value: '3 moves for stronger knees.' } });
  fireEvent.click(runButton());

  expect(await screen.findByText('Niche hit', {}, { timeout: 4000 })).toBeInTheDocument();
  const body = (fetchMock.mock.calls[0] as unknown as [string, RequestInit])[1].body as FormData;
  expect(body.get('niche')).toBe('fitness');
  expect(screen.getByText('Confidence 80%')).toBeInTheDocument();
  expect(screen.getByText('Add a concrete number.')).toBeInTheDocument();
  expect(screen.getByText('nutrition')).toBeInTheDocument();
});
