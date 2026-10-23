import { make0, type Box0 } from './file0.ts'

export interface Box15<T> {
	value: T
	map<U>(fn: (value: T) => U): Box15<U>
}

export function make15<T>(value: T): Box15<T> {
	return { value, map: (fn) => make15(fn(value)) }
}

export class Store15<K extends string, V> {
	private readonly items = new Map<K, V>()
	set(key: K, value: V): this {
		this.items.set(key, value)
		return this
	}
	get(key: K): V | undefined {
		return this.items.get(key)
	}
	entries(): Array<[K, V]> {
		return [...this.items.entries()]
	}
}

export function use15(other: Box0<number>) {
	const store = new Store15<'a' | 'b', Box0<number>>()
	store.set('a', other).set('b', make0(other.value + 15))
	const mapped = make15(other.value)
		.map((n) => `${n}`)
		.map((s) => s.length)
	const total = store
		.entries()
		.reduce((sum, [, box]) => sum + box.value, mapped.value)
	const pairs = Object.entries({ x: total, y: mapped.value }).map(([k, v]) => ({
		k,
		v,
	}))
	return pairs.filter(({ v }) => v > 0).map(({ k }) => k.toUpperCase())
}
