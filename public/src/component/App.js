import { html, render, useEffect, useState } from 'https://unpkg.com/htm@3.1.0/preact/standalone.module.js'
import { Event } from './Event.js'
import { Header } from './Header.js'
import { Filter, filterFunc } from './Filter.js'

function App() {
    const [event, setEvent] = useState(null);
    const [events, setEvents] = useState([]);
    const [isStreaming, setStreaming] = useState(true);
    const [automaticScrollEnabled, setAutomaticScrollEnabled] = useState(false);

    const [textFilter, setTextFilter] = useState(null);
    const [operationFilter, setOperationFilter] = useState(null);
    const [limit, setLimit] = useState(100);

    useEffect(() => {
        const eventSource = new EventSource('/sse/event');

        eventSource.addEventListener('open', () => setStreaming(true));
        eventSource.addEventListener('error', () => setStreaming(false));
        eventSource.addEventListener('event', (e) => setEvent(JSON.parse(e.data)));
    }, []);

    useEffect(() => {
        // Keep maximum 500 events in memory.
        const filteredEvents = events.filter(event => event !== null).slice(-500);
        setEvents([...filteredEvents, event]);

        if (automaticScrollEnabled) {
            window.scrollTo(0, document.body.scrollHeight);
        }
    }, [event]);

    return html`
        <${Header} />

        <div class="px-8 mb-10">
            <${Filter}
                onTextFilter=${setTextFilter}
                onOperationFilter=${setOperationFilter}
                onLimit=${setLimit}
            />

            <hr class="mt-4" />

            <div class="mt-4 space-y-2">
                ${event ? (
                    events.slice(-limit).filter(filterFunc(textFilter, operationFilter)).map((event, key) => (
                        html`<${Event} event=${event} key=${key} />`
                    ))
                ) : html`<div class="text-center text-xs uppercase">No activity at the moment 😴</div>`}
            </div>

            <div class="mt-8">
                ${isStreaming ? (
                    html`<label class="mt-0.5 inline-flex items-center text-xs uppercase float-right">
                            <input type="checkbox" class="h-4 w-4" ${automaticScrollEnabled ? 'checked' : null} onClick=${() => { setAutomaticScrollEnabled(!automaticScrollEnabled); }} />
                            <span class="ml-1 text-gray-500">Automatic scroll</span>
                        </label>
                        <div class="mt-4 space-y-2 text-gray-500 text-xs uppercase">
                            Watching collection changes...
                            <svg class="inline animate-spin ml-1 h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                            </svg>
                        </div>`
                ) : (
                    html`<div class="mt-4 space-y-2 text-red-500 text-xs uppercase">
                        Unable to connect or connection lost. Refresh the page to try reconnecting.
                    </div>`
                )}
            </div>
        </div>
    `
}

render(html`<${App} />`, document.getElementById('app'))