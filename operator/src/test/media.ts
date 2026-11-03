/** Controllable viewport + media-query state for layout tests. */
export type Viewport = 'wide' | 'narrow';

let viewport: Viewport = 'wide';

export function setNarrowViewport(next: Viewport): void {
  viewport = next;
}

export function currentViewport(): Viewport {
  return viewport;
}
