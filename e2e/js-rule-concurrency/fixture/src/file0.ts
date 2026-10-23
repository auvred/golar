import { make1, type Box1 } from './file1.ts'

export interface Box0<T> {
	value: T
	map<U>(fn: (value: T) => U): Box0<U>
}

export function make0<T>(value: T): Box0<T> {
	return { value, map: (fn) => make0(fn(value)) }
}

export class Store0<K extends string, V> {
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

export function use0(other: Box1<number>) {
	const store = new Store0<'a' | 'b', Box1<number>>()
	store.set('a', other).set('b', make1(other.value + 0))
	const mapped = make0(other.value)
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
