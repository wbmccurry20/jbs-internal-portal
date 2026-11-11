import { describe, expect, it } from 'vitest';
import { filterLicenses } from '../utils/licenses';

const records = [
  { state: 'AL', city: 'Hueytown', license_type: 'Business', license_number: '18396.1', entity_name: 'JBS', expiration_date: '2026-12-31', status: 'expiring', is_city_license: true },
  { state: 'AL', city: null, license_type: 'Contractor', license_number: '60614', entity_name: 'JBS', expiration_date: '2026-12-01', status: 'expiring', is_city_license: false },
  { state: 'GA', city: null, license_type: 'Business', license_number: null, entity_name: null, expiration_date: null, status: 'unknown', is_city_license: false },
];
const states = { AL: 'Alabama', GA: 'Georgia' };

describe('license search', () => {
  it('filters by city, status and jurisdiction', () => {
    expect(filterLicenses(records, { search: ' HUEYTOWN ', status: 'expiring', jurisdiction: 'city' }, states)).toEqual([records[0]]);
    expect(filterLicenses(records, { search: 'Alabama', status: 'expiring', jurisdiction: 'state' }, states)).toEqual([records[1]]);
    expect(filterLicenses(records, { search: '18396.1', status: 'expired', jurisdiction: 'all' }, states)).toEqual([]);
  });

  it('orders dated licenses first by expiration and leaves missing dates last', () => {
    expect(filterLicenses(records, { search: '', status: 'all', jurisdiction: 'all' }, states)).toEqual([records[1], records[0], records[2]]);
  });
});
