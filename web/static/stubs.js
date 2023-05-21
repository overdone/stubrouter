function createStubFormElement(target, formData, isNew= false) {
    return `
        <li class="stub">
            <form isnew=${isNew}>
                <div class="stub-head">
                    <input name="path" type="text" class="stub-name" ${isNew ? '' : 'readonly'} value="${formData.path || ''}" />
                    <div class="head-controls">
                        <input type="button" value="Save" class="button" onClick="onSaveStubClick(this, '${target}')" tabindex="0" />
                        <input type="button" value="Remove" class="button remove-button" onClick="onRemoveStubClick(this, '${target}')" tabindex="0" />
                    </div>
                </div>
                <input name="code" type="number" class="number" placeholder="Code" value="${formData.code || ''}" />
                <textarea name="headers" rows="2" placeholder="Headers">${JSON.stringify(formData.headers) || '{}'}</textarea>
                <textarea name="data" rows="5" placeholder="Data">${formData.data || ''}</textarea>
                <input name="timeout" type="number" class="number" placeholder="Timeout, ms" value="${formData.timeout || ''}" />
            </form>
        </li>`;
}


async function onSaveStubClick(el, target) {
    const stubForm = el.closest('form');
    const formData = Object.fromEntries(new FormData(stubForm));

    try {
        const resp = await fetch(
        `/stubapi/?${new URLSearchParams({ target, path: formData.path })}`,
        {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(formData),
            },
        );
        if (resp.status === 200) {
            stubForm.setAttribute('isnew', "false");
            stubForm.querySelector('[name="path"]').setAttribute('readonly', '');
        }
    } catch (e) {
        console.log(e);
    }
}


async function onRemoveStubClick(el, target) {
    const stubForm = el.closest('form');
    const formData = Object.fromEntries(new FormData(stubForm));

    try {
        if (stubForm.getAttribute('isnew') === 'false') {
            const resp = await fetch(
            `/stubapi/?${new URLSearchParams({ target, path: formData.path })}`,
            { method: 'DELETE' },
            );
            if (resp.status === 200) {
                stubForm.remove();
            }
        } else {
            stubForm.remove();
        }
    } catch (e) {
        console.log(e);
    }
}


function onAddStubClick() {
    const params = new URLSearchParams(window.location.search);
    const target = params.get('target');
    const stubList = document.querySelector('.list.stubs');
    stubList.insertAdjacentHTML('beforeend', createStubFormElement(target, {}, true));
}


async function onOpenStubsPage() {
    const params = new URLSearchParams(window.location.search);
    const target = params.get('target');
    if (!target) return;

    const title = document.querySelector('.title');
    title.innerText = target;

    try {
        const resp = await fetch(`/stubapi/?${new URLSearchParams({ target })}`);
        const data = await resp.json();
        const stubList = document.querySelector('.list.stubs');

        if (!stubList) return;

        Object.entries(data).forEach((s) => {
            const formData = { path: s[0], ...s[1] };
            stubList.insertAdjacentHTML('beforeend', createStubFormElement(target, formData));
        })
    } catch (e) {
        console.log(e);
    }
}


onOpenStubsPage().then().catch();



