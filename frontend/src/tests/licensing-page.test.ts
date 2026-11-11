import { expect, it, vi } from 'vitest';
import licensingPage from '../pages/licensing.astro?raw';
import { escapeHtml } from '../utils/html';
import * as licenses from '../utils/licenses';

it('opens search results, handles document errors, and clears filters', async () => {
  const parsed = new DOMParser().parseFromString(licensingPage, 'text/html');
  document.body.innerHTML = parsed.body.innerHTML;
  const dialog = document.getElementById('license-detail-dialog') as HTMLDialogElement;
  dialog.showModal = () => { dialog.setAttribute('open', ''); };
  dialog.close = () => {
    dialog.removeAttribute('open');
    dialog.dispatchEvent(new Event('close'));
  };
  vi.spyOn(localStorage, 'getItem').mockReturnValue('test-token');
  const records = [
    { id: 1, state: 'AL', city: 'Hueytown', is_city_license: true, license_type: 'Business', license_number: '123', entity_name: 'JBS', issue_date: null, expiration_date: '2026-12-31', status: 'expiring', renewal_fee: 0, notes: '', document_count: 1 },
    { id: 2, state: 'GA', city: '', is_city_license: false, license_type: 'Contractor', license_number: '', entity_name: '', issue_date: null, expiration_date: null, status: 'unknown', renewal_fee: 0, notes: '', document_count: 0 },
  ];
  let documentAttempts = 0;
  const fetchMock = vi.fn(async (url: string) => {
    if (url.endsWith('/sharepoint/status')) return { ok: false };
    if (url.endsWith('/licenses')) return { ok: true, json: async () => records };
    if (url.endsWith('/licenses/1/documents')) {
      return { ok: true, json: async () => [{ document_name: 'License.pdf', document_type: 'certificate', document_url: 'https://example.com/license.pdf' }] };
    }
    if (url.endsWith('/licenses/2/documents')) {
      documentAttempts++;
      if (documentAttempts === 1) return { ok: false };
      if (documentAttempts === 2) {
        return { ok: true, json: async () => [{ document_name: 'Bad link', document_type: 'other', document_url: 'javascript:alert(1)' }] };
      }
      return { ok: true, json: async () => [] };
    }
    throw new Error(`Unexpected request: ${url}`);
  });
  vi.stubGlobal('fetch', fetchMock);
  const script = licensingPage.match(/<script>([\s\S]*?)<\/script>/)?.[1];
  if (!script) throw new Error('Licensing script not found');
  // esbuild needs encoded bytes from the same realm as jsdom's Uint8Array.
  const NativeTextEncoder = globalThis.TextEncoder;
  class TestTextEncoder extends NativeTextEncoder {
    encode(input?: string) {
      return new Uint8Array(super.encode(input));
    }
  }
  vi.stubGlobal('TextEncoder', TestTextEncoder);
  try {
    const { transformSync } = await import('esbuild');
    const compiled = transformSync(script, { loader: 'ts', format: 'cjs', target: 'es2020' }).code;
    const modules: Record<string, object> = {
      '../utils/html': { escapeHtml },
      '../utils/api': { getApiUrl: () => '/api' },
      '../utils/toast': { toast: { error: vi.fn(), success: vi.fn(), warning: vi.fn() } },
      '../utils/licenses': licenses,
    };
    new Function('require', 'exports', compiled)((name: string) => {
      if (!modules[name]) throw new Error(`Unexpected import: ${name}`);
      return modules[name];
    }, {});
    await vi.waitFor(() => expect(document.querySelector('[data-result-license-id="1"]')).not.toBeNull());
    expect(document.querySelector('[data-result-license-id="2"]')).toBeNull();
    const first = document.querySelector<HTMLElement>('[data-result-license-id="1"]')!;
    first.querySelector<HTMLButtonElement>('[data-license-action="edit"]')!.click();
    expect(dialog.open).toBe(false);
    expect(document.getElementById('license-modal')!.classList.contains('hidden')).toBe(false);
    (document.getElementById('cancel-btn') as HTMLButtonElement).click();
    first.click();
    await vi.waitFor(() => expect(dialog.querySelector('a')?.textContent).toBe('License.pdf'));
    expect(dialog.open).toBe(true);
    expect(dialog.querySelector('a')?.getAttribute('href')).toBe('https://example.com/license.pdf');
    expect(dialog.textContent).toContain('Hueytown, Alabama');
    (document.getElementById('license-detail-edit') as HTMLButtonElement).click();
    expect(dialog.open).toBe(false);
    expect(document.getElementById('license-modal')!.classList.contains('hidden')).toBe(false);
    (document.getElementById('cancel-btn') as HTMLButtonElement).click();

    (document.getElementById('clear-license-filters') as HTMLButtonElement).click();
    const second = document.querySelector<HTMLElement>('[data-result-license-id="2"]')!;
    expect(second.textContent).toContain('Expiration date needed');
    second.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
    await vi.waitFor(() => expect(document.getElementById('retry-license-documents')).not.toBeNull());
    (document.getElementById('retry-license-documents') as HTMLButtonElement).click();
    await vi.waitFor(() => expect(dialog.textContent).toContain('Bad link'));
    expect(dialog.textContent).toContain('invalid link');
    expect(dialog.querySelector('a')).toBeNull();
    (document.getElementById('license-detail-close') as HTMLButtonElement).click();
    second.querySelector<HTMLButtonElement>('[data-license-action="view"]')!.click();
    await vi.waitFor(() => expect(dialog.textContent).toContain('No documents linked to this license.'));
  } finally {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  }
});
