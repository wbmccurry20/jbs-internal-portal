import { describe, expect, it, vi } from 'vitest';
import { bindLicenseSummaryFilters, filterLicenses, getExpirationTiming } from '../utils/licenses';
import licensingPage from '../pages/licensing.astro?raw';

const records = [
  { state: 'AL', city: 'Hueytown', license_type: 'Business', license_number: '18396.1', entity_name: 'JBS', expiration_date: '2026-12-31', status: 'expiring', is_city_license: true },
  { state: 'AL', city: null, license_type: 'Contractor', license_number: '60614', entity_name: 'JBS', expiration_date: '2026-12-01', status: 'expiring', is_city_license: false },
  { state: 'GA', city: null, license_type: 'Business', license_number: null, entity_name: null, expiration_date: null, status: 'unknown', is_city_license: false },
];
const states = { AL: 'Alabama', GA: 'Georgia' };

describe('expiration timing', () => {
  it('uses the New York calendar date, including across daylight saving time', () => {
    const now = new Date('2026-03-08T04:30:00Z');
    expect(getExpirationTiming('2026-03-07', now)).toBe('Expires today');
    expect(getExpirationTiming('2026-03-08', now)).toBe('Expires in 1 day');
    expect(getExpirationTiming('2026-03-09', now)).toBe('Expires in 2 days');
    expect(getExpirationTiming('2026-03-06', now)).toBe('Expired 1 day ago');
    expect(getExpirationTiming(null, now)).toBe('Expiration date needed');
  });
});

describe('license search', () => {
  it('filters by city, status and jurisdiction', () => {
    expect(filterLicenses(records, { search: ' HUEYTOWN ', status: 'expiring', jurisdiction: 'city' }, states)).toEqual([records[0]]);
    expect(filterLicenses(records, { search: 'Alabama', status: 'expiring', jurisdiction: 'state' }, states)).toEqual([records[1]]);
    expect(filterLicenses(records, { search: '18396.1', status: 'expired', jurisdiction: 'all' }, states)).toEqual([]);
  });

  describe('license summary shortcuts', () => {
    it('opens the matching results for every card and clears other filters', () => {
      const page = new DOMParser().parseFromString(licensingPage, 'text/html');
      const heading = page.querySelector<HTMLElement>('#find-licenses-heading')!;
      const search = page.querySelector<HTMLInputElement>('#license-search')!;
      const status = page.querySelector<HTMLSelectElement>('#license-status-filter')!;
      const jurisdiction = page.querySelector<HTMLSelectElement>('#license-jurisdiction-filter')!;
      const buttons = page.querySelectorAll<HTMLButtonElement>('[data-license-status]');
      expect(buttons).toHaveLength(6);
      expect([...buttons].every(button => button.disabled)).toBe(true);

      document.body.replaceChildren(heading, search, status, jurisdiction, ...buttons);
      heading.scrollIntoView = vi.fn();
      const focus = vi.spyOn(heading, 'focus');
      const examples = [
        ...records,
        { ...records[0], status: 'active' },
        { ...records[0], status: 'pending' },
        { ...records[0], status: 'expired' },
      ];
      let results = examples;
      const render = vi.fn(() => {
        results = filterLicenses(examples, { search: search.value, status: status.value, jurisdiction: jurisdiction.value }, states);
      });
      bindLicenseSummaryFilters(buttons, search, status, jurisdiction, heading, render);
      const expected: Record<string, number> = { all: 6, active: 1, expiring: 2, pending: 1, expired: 1, unknown: 1 };
      for (const button of buttons) {
        button.disabled = false;
        search.value = 'unmatched search';
        jurisdiction.value = 'city';
        button.click();
        const selectedStatus = button.dataset.licenseStatus!;
        expect(status.value).toBe(selectedStatus);
        expect(search.value).toBe('');
        expect(jurisdiction.value).toBe('all');
        expect(results).toHaveLength(expected[selectedStatus]);
        expect(results.every(record => selectedStatus === 'all' || record.status === selectedStatus)).toBe(true);
      }
      expect(render).toHaveBeenCalledTimes(6);
      expect(focus).toHaveBeenCalledTimes(6);
      expect(heading.scrollIntoView).toHaveBeenCalledTimes(6);
    });
  });

  it('orders dated licenses first by expiration and leaves missing dates last', () => {
    expect(filterLicenses(records, { search: '', status: 'all', jurisdiction: 'all' }, states)).toEqual([records[1], records[0], records[2]]);
  });
});
