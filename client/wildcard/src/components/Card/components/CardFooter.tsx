import { forwardRef } from 'react'

import classNames from 'classnames'

import { ForwardReferenceComponent } from '../../..'

import styles from './CardFooter.module.scss'

interface CardFooterProps {}

export const CardFooter = forwardRef(({ as: Component = 'div', children, className, ...attributes }, reference) => (
    <Component ref={reference} className={classNames(styles.cardFooter, className)} {...attributes}>
        {children}
    </Component>
)) as ForwardReferenceComponent<'div', CardFooterProps>
