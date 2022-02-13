import { html } from 'https://unpkg.com/htm@3.1.0/preact/standalone.module.js'

export const Event = ({ event }) => {
    const value = atob(event.value);
    const date = new Intl.DateTimeFormat('default', {
        year: 'numeric', month: 'numeric', day: 'numeric',
        hour: 'numeric', minute: 'numeric', second: 'numeric',
        hour12: false,
    }).format(new Date(event.timestamp * 1000));

    let operationColor = 'bg-green-500';
    let operationBackgroundColor = 'bg-green-50';

    switch (event.operation) {
        case 'update':
            operationColor = 'bg-yellow-500';
            operationBackgroundColor = 'bg-yellow-50';
            break;
        case 'delete':
            operationColor = 'bg-red-500';
            operationBackgroundColor = 'bg-red-50';
            break;
    }

    return html`
    <div class="flex rounded-md shadow transform duration-100 hover:scale-[1.02] hover:${operationBackgroundColor}">
        <div class="flex flex-wrap items-center">
            <div class="px-2 text-xs text-gray-700">
                ${date}
            </div>
            <div class="w-24 px-2 py-1 text-xs text-center font-bold rounded text-white uppercase ${operationColor}">
                ${event.operation}
            </div>
            <div class="px-2 py-2 text-xs font-bold">
                ${event.id}
            </div>
            <div class="px-2 py-2 text-xs">
                ${value}
            </div>
        </div>
    </div>
`
};