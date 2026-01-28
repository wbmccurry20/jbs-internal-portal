// Loading skeleton components and utilities

export function createCardSkeleton(): HTMLElement {
  const skeleton = document.createElement('div');
  skeleton.className = 'bg-white overflow-hidden shadow rounded-lg animate-pulse';
  skeleton.innerHTML = `
    <div class="px-4 py-5 sm:p-6">
      <div class="h-4 bg-gray-200 rounded w-1/2 mb-2"></div>
      <div class="h-8 bg-gray-200 rounded w-3/4"></div>
    </div>
  `;
  return skeleton;
}

export function createTableSkeleton(rows: number = 5): HTMLElement {
  const skeleton = document.createElement('div');
  skeleton.className = 'bg-white shadow rounded-lg overflow-hidden animate-pulse';
  
  const rowsHtml = Array.from({ length: rows }, () => `
    <div class="border-b border-gray-200 px-6 py-4">
      <div class="flex gap-4">
        <div class="h-4 bg-gray-200 rounded flex-1"></div>
        <div class="h-4 bg-gray-200 rounded flex-1"></div>
        <div class="h-4 bg-gray-200 rounded flex-1"></div>
      </div>
    </div>
  `).join('');
  
  skeleton.innerHTML = `
    <div class="px-6 py-4 border-b border-gray-200">
      <div class="h-5 bg-gray-200 rounded w-1/4"></div>
    </div>
    ${rowsHtml}
  `;
  return skeleton;
}

export function createStatsSkeleton(): HTMLElement {
  const skeleton = document.createElement('div');
  skeleton.className = 'grid grid-cols-1 md:grid-cols-3 gap-6';
  skeleton.innerHTML = `
    <div class="bg-white overflow-hidden shadow rounded-lg animate-pulse">
      <div class="px-4 py-5 sm:p-6">
        <div class="h-4 bg-gray-200 rounded w-2/3 mb-2"></div>
        <div class="h-8 bg-gray-200 rounded w-1/2"></div>
      </div>
    </div>
    <div class="bg-white overflow-hidden shadow rounded-lg animate-pulse">
      <div class="px-4 py-5 sm:p-6">
        <div class="h-4 bg-gray-200 rounded w-2/3 mb-2"></div>
        <div class="h-8 bg-gray-200 rounded w-1/2"></div>
      </div>
    </div>
    <div class="bg-white overflow-hidden shadow rounded-lg animate-pulse">
      <div class="px-4 py-5 sm:p-6">
        <div class="h-4 bg-gray-200 rounded w-2/3 mb-2"></div>
        <div class="h-8 bg-gray-200 rounded w-1/2"></div>
      </div>
    </div>
  `;
  return skeleton;
}

export function showLoading(containerId: string, type: 'card' | 'table' | 'stats' = 'card'): void {
  const container = document.getElementById(containerId);
  if (!container) return;
  
  container.innerHTML = '';
  
  let skeleton: HTMLElement;
  switch (type) {
    case 'stats':
      skeleton = createStatsSkeleton();
      break;
    case 'table':
      skeleton = createTableSkeleton();
      break;
    default:
      skeleton = createCardSkeleton();
  }
  
  container.appendChild(skeleton);
}

export function hideLoading(containerId: string): void {
  const container = document.getElementById(containerId);
  if (!container) return;
  container.innerHTML = '';
}

// Inline spinner for buttons
export function createSpinner(size: 'sm' | 'md' | 'lg' = 'md'): HTMLElement {
  const sizes = {
    sm: 'w-4 h-4',
    md: 'w-5 h-5',
    lg: 'w-6 h-6'
  };
  
  const spinner = document.createElement('svg');
  spinner.className = `animate-spin ${sizes[size]}`;
  spinner.setAttribute('xmlns', 'http://www.w3.org/2000/svg');
  spinner.setAttribute('fill', 'none');
  spinner.setAttribute('viewBox', '0 0 24 24');
  spinner.innerHTML = `
    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
  `;
  return spinner;
}

// Button loading state helper
export function setButtonLoading(button: HTMLButtonElement, loading: boolean): void {
  if (loading) {
    button.disabled = true;
    button.dataset.originalText = button.innerHTML;
    const spinner = createSpinner('sm');
    button.innerHTML = '';
    button.appendChild(spinner);
    button.appendChild(document.createTextNode(' Loading...'));
  } else {
    button.disabled = false;
    if (button.dataset.originalText) {
      button.innerHTML = button.dataset.originalText;
      delete button.dataset.originalText;
    }
  }
}
