import { html, useEffect, useState } from 'https://unpkg.com/htm@3.1.0/preact/standalone.module.js'

export const Filter = ({
    onTextFilter = () => {},
    onOperationFilter = () => {},
    onLimit = () => {},
}) => {
    const [textFilter, setTextFilter] = useState(null);
    const [operationFilter, setOperationFilter] = useState(null);
    const [limit, setLimit] = useState(100);

    useEffect(() => onTextFilter(textFilter), [textFilter]);
    useEffect(() => onOperationFilter(operationFilter), [operationFilter]);
    useEffect(() => onLimit(limit), [limit]);

    return html`
        <div class="flex flex-wrap gap-4 text-xs text-gray-500 uppercase">
            <label class="mt-0.5 inline-flex items-center">
                <span class="mr-2">Query</span>
                <input class="px-2 py-1 border rounded-xl" type="text" onKeyUp="${e => setTextFilter(e.target.value)}" />
            </label>

            <label class="mt-0.5 inline-flex items-center">
                <span class="mr-2">Operation</span>
                <select onChange=${e => setOperationFilter(e.target.value)} class="px-3 py-1.5 uppercase border border-solid border-gray-200 rounded-full">
                    <option selected>All</option>
                    <option value="insert">Insert</option>
                    <option value="update">Update</option>
                    <option value="delete">Delete</option>
                </select>
            </label>

            <label class="mt-0.5 inline-flex items-center">
                <span class="mr-2">Keep last</span>
                <select onChange=${e => setLimit(e.target.value)} class="px-3 py-1.5 uppercase border border-solid border-gray-200 rounded-full">
                    <option value="10">10</option>
                    <option value="50">50</option>
                    <option selected value="100">100</option>
                    <option value="200">200</option>
                    <option value="300">300</option>
                    <option value="500">500</option>
                </select>
                <span class="ml-2">events</span>
            </label>
        </div>
    `
}

export const filterFunc = (textFilter, operationFilter) => event => {
    if (event === null) {
        return false;
    }

    let keep = true;

    if (textFilter
        && !event.id.includes(textFilter)
        && !atob(event.value).includes(textFilter)) {
        keep = false;
    }

    if (operationFilter && event.operation !== operationFilter) {
        keep = false;
    }

    return keep;
};