interface SearchableLicense {
  state: string;
  city: string | null;
  license_type: string;
  license_number: string | null;
  entity_name: string | null;
  expiration_date: string | null;
  status: string;
  is_city_license: boolean;
}

export function filterLicenses<T extends SearchableLicense>(
  licenses: T[],
  filters: { search: string; status: string; jurisdiction: string },
  stateNames: Record<string, string>,
): T[] {
  const search = filters.search.trim().toLowerCase();
  return licenses.filter(license => {
    const searchable = [license.state, stateNames[license.state], license.city, license.license_type, license.license_number, license.entity_name]
      .filter(Boolean).join(' ').toLowerCase();
    return searchable.includes(search)
      && (filters.status === 'all' || license.status === filters.status)
      && (filters.jurisdiction === 'all' || (filters.jurisdiction === 'city') === license.is_city_license);
  }).sort((a, b) => (a.expiration_date || '9999-12-31').localeCompare(b.expiration_date || '9999-12-31')
    || a.state.localeCompare(b.state));
}
